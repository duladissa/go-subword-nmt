package bpe

import (
	"bufio"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/duladissa/go-subword-nmt/utils"
	"github.com/pkg/errors"
)

const (
	//EndWord ... End word
	EndWord = "</w>"
	//EndWordLength ... End word Length
	EndWordLength = 4
	//TokenDelim ... Token delim
	TokenDelim = "@@"
	//TokenDelimLength ... Token delim length
	TokenDelimLength = 2
	cacheMaxEntries  = 1000
)

//BPE ... BPE struct
type BPE struct {
	codes           map[utils.Pair]int
	vocab           map[string]int
	reversedCodes   map[string]utils.Pair
	glossaries      map[string]int
	glossariesRegex string
	cache           map[string][]string
	separator       string
	codesPath       string
	vocabPath       string
}

type symbol struct {
	i    int
	pair utils.Pair
}

//NewBPE ... Create new BPE from codes and vocab
func NewBPE(codesPath string, vocabPath string) (*BPE, error) {
	bpe := &BPE{
		codes:         map[utils.Pair]int{},
		vocab:         map[string]int{},
		reversedCodes: map[string]utils.Pair{},
		codesPath:     codesPath,
		vocabPath:     vocabPath,
		separator:     TokenDelim,
		cache:         map[string][]string{},
	}
	bpe.readVocab()
	err := bpe.readCodes()
	if err != nil {
		return nil, errors.Wrap(err, "Unable to create BPE")
	}
	return bpe, nil
}

func (bpe *BPE) readVocab() {
	f, err := os.OpenFile(bpe.vocabPath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		splits := strings.Split(sc.Text(), " ")
		bpe.vocab[splits[0]], _ = strconv.Atoi(splits[1])
	}
}

func (bpe *BPE) readCodes() error {
	f, err := os.OpenFile(bpe.codesPath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "Cannot open codes file")
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.Contains(line, "#version:") {
			continue
		}
		splits := strings.Split(line, " ")
		pair := utils.Pair{X: splits[0], Y: splits[1]}
		bpe.codes[pair] = len(bpe.codes)
		bpe.reversedCodes[splits[0]+splits[1]] = pair
	}
	return nil
}

//ProcessLine ... Processing line
//segment line, dealing with leading and trailing whitespace
func (bpe *BPE) ProcessLine(line string, dropout int) string {
	if len(bpe.cache) > cacheMaxEntries {
		bpe.cache = map[string][]string{}
	}
	out := ""
	leadingWhitespace := len(line) - len(strings.TrimLeft(line, "\r\n "))
	if leadingWhitespace > 0 {
		out = line[:leadingWhitespace]
	}
	out += bpe.segment(line, dropout)

	trailingWhitespace := len(line) - len(strings.TrimRight(line, "\r\n "))
	if trailingWhitespace > 0 && trailingWhitespace != len(line) {
		out += line[-trailingWhitespace:]
	}
	return out
}

//segment single sentence (whitespace-tokenized string) with BPE encoding
func (bpe *BPE) segment(sentence string, dropout int) string {
	sentence = strings.Trim(sentence, "\r\n ")
	tokens := strings.Fields(sentence)
	segments := bpe.segmentTokens(tokens, dropout)
	return strings.Join(segments, " ")
}

//segment a sequence of tokens with BPE encoding
func (bpe *BPE) segmentTokens(tokens []string, dropout int) []string {
	output := make([]string, 0)
	for _, word := range tokens {
		newWord := make([]string, 0)
		for _, segment := range bpe.isolateGlossaries(word) {
			for _, out := range bpe.encode(segment, dropout) {
				newWord = append(newWord, out)
			}
		}
		for _, item := range newWord[:len(newWord)-1] {
			output = append(output, (item + bpe.separator))
		}
		output = append(output, newWord[len(newWord)-1])
	}
	return output
}

//TODO implementation
func (bpe *BPE) isolateGlossaries(word string) []string {
	wordSegments := []string{word}
	/*
			def _isolate_glossaries(self, word):
		        word_segments = [word]
		        for gloss in self.glossaries:
		            word_segments = [out_segments for segment in word_segments
		                                 for out_segments in isolate_glossary(segment, gloss)]
		        return word_segments
	*/
	return wordSegments
}

//Encode word based on list of BPE merge operations, which are applied consecutively
func (bpe *BPE) encode(orig string, dropout int) []string {
	//TODO
	if dropout == 0 {
		val, ok := bpe.cache[orig]
		if ok {
			return val
		}
	}
	// if not dropout and orig in cache:
	//     return cache[orig]
	// if glossaries_regex and glossaries_regex.match(orig):
	//     cache[orig] = (orig,)
	//     return (orig,)

	// if len(orig) == 1:
	//     return orig

	// if version == (0, 1):
	//     word = list(orig) + ['</w>']
	// elif version == (0, 2): # more consistent handling of word-final segments
	//     word = list(orig[:-1]) + [orig[-1] + '</w>']
	// else:
	// 	raise NotImplementedError
	//Only supporting version 2.0
	firstPart := orig[:len(orig)-1]
	lastPart := orig[len(orig)-1:] + "</w>"

	word := make([]string, 0)
	for _, char := range firstPart {
		word = append(word, string(char))
	}
	word = append(word, lastPart)

	for len(word) > 1 {
		//get list of symbol pairs; optionally apply dropout
		pairs := bpe.getPairsFromTheWord(word, dropout)
		if len(pairs) == 0 {
			break
		}
		//get first merge operation in list of BPE codes
		_, symbol := bpe.min(pairs)
		bigram := symbol.pair

		//find start position of all pairs that we want to merge
		possitions := make([]int, 0)
		for _, value := range pairs {
			//value is type of symbol
			if value.pair == bigram {
				possitions = append(possitions, value.i)
			}
		}

		i := 0
		newWord := make([]string, 0)
		bigramStr := bigram.X + bigram.Y
		for _, j := range possitions {
			//merges are invalid if they start before current position. This can happen if there are overlapping pairs: (x x x -> xx x)
			if j < i {
				continue
			}
			newWord = append(newWord, word[i:j]...)
			newWord = append(newWord, bigramStr)
			i = j + 2 //continue after merged pair
		}
		newWord = append(newWord, word[i:]...) //add all symbols until end of word
		word = newWord
	}

	//don't print end-of-word symbols
	lastWord := word[len(word)-1]
	if lastWord == "</w>" {
		word = word[:len(word)-1]
	} else if strings.HasSuffix(lastWord, "</w>") {
		word[len(word)-1] = lastWord[:len(lastWord)-4]
	}

	// word = tuple(word)
	// if vocab:
	//     word = check_vocab_and_split(word, bpe_codes_reverse, vocab, separator)
	bpe.cache[orig] = word
	// return word

	return word
}

//getPairsFromTheWord : Implementation of below expression
//pairs = [(bpe_codes[pair],i,pair) for (i,pair) in enumerate(zip(word, word[1:])) if (not dropout or random.random() > dropout) and pair in bpe_codes]
func (bpe *BPE) getPairsFromTheWord(word []string, dropout int) map[int]symbol {
	//get list of symbol pairs; optionally apply dropout
	zipEnumerated := utils.ZipAndEmumerateTwoArrays(word, word[1:])
	pairs := make(map[int]symbol)
	for i, pair := range zipEnumerated {
		val, ok := bpe.codes[pair]
		if (dropout == 0 || rand.Float64() > float64(dropout)) && ok {
			pairs[val] = symbol{i: i, pair: pair}
		}
	}
	return pairs
}

func (bpe *BPE) min(items map[int]symbol) (int, symbol) {
	keys := make([]int, 0)
	for k := range items {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	return keys[0], items[keys[0]]
}
