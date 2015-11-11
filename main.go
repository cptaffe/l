package main

import (
  "fmt"
  "os"
  "bufio"
)

// Match-Nexter interface
// Allows for magic to happen inside the type.
type MatchNexter interface {
  // Returns next possible states
  Match(rune) []MatchNexter
}

// Matching function used by State type
type StateMatcherFunc func(rune) []MatchNexter

// State type implements MatchNexter
type State struct {
  // Matcher Function returns true on a match.
  Matcher StateMatcherFunc
}

// Returns true on matching rune
func (s *State) Match(r rune) []MatchNexter {
  return s.Matcher(r)
}

type Lexer struct {
  States []MatchNexter
  Hooks []Lexer
  Matched string
}

type Match struct {
   Success bool
   Match string
}

// Uses an array of possible MatchNexters
// Each MatchNexters' Match function is called
// exactly once per set of MatchNexters.
// If Match returns true, then Next is called.
func (l *Lexer) Lex(runes <-chan rune) chan Match {
  out := make(chan Match)
  go func(runes <-chan rune) {
    defer close(out)
    for r := range runes {
      var nextStates []MatchNexter
      for _, s := range l.States {
        if s != nil {
          nextStates = append(nextStates, s.Match(r)...)
        } else {
          // Have encountered a possible exit condition,
          // place a hook.
          l.Hooks = append([]Lexer{ *l }, l.Hooks...)
        }
      }
      l.States = nextStates
      if len(l.States) == 0 {
        if len(l.Hooks) > 0 {
          out <- Match{ Success: true, Match: l.Hooks[0].Matched }
        } else {
          out <- Match{ Success: false, Match: l.Matched }
        }
        return // No more possible states.
      } else {
        l.Matched += string(r)
      }
    }
  }(runes)
  return out
}

type RuneMatcher struct {
  MatchRune rune
  Nexts []MatchNexter
}

func (rm *RuneMatcher) Match(r rune) []MatchNexter {
  if r == rm.MatchRune {
    return rm.Nexts
  } else {
    return []MatchNexter{}
  }
}

// Creates chain of RuneMatchers to match
// an entire string.
func StringMatcher(match string) (first, last *RuneMatcher) {
  for _, r := range match {
    rm := &RuneMatcher{ MatchRune: r }
    if last != nil {
      last.Nexts = []MatchNexter{ MatchNexter(rm) }
    } else {
      first = rm
    }
    last = rm
  }
  return
}

func main() {
  runes := make(chan rune)

  // Generate loop to beginning of series,
  // acts like (ab)* in regex.
  first, last := StringMatcher("ab")
  last.Nexts = []MatchNexter{ first, nil }

  l := Lexer{
    States: []MatchNexter{ MatchNexter(first) },
  }

  // Asynchrounously lex input
  out := l.Lex(runes)

  // Asynchrounously create input and close channel.
  // On close of channel, l.Lex will close out.
  go func() {
    line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
    for _, r := range line {
      runes <- r
    }
    close(runes)
  }()

  // Keep-alive until out is closed.
  for s := range out {
    if s.Success {
      fmt.Println("Found match '"+s.Match+"'")
    } else {
      fmt.Println("No match found, got: "+s.Match);
    }
  }
}
