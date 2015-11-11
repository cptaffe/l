package main

import (
  "fmt"
  "os"
  "bufio"
  "unicode"
)

// Match-Nexter interface
// Allows for magic to happen inside the type.
type Matcher interface {
  // Returns next possible states
  Match(rune) []Matcher
}

// Matching function used by State type
type StateMatcherFunc func(rune) []Matcher

// State type implements MatchNexter
type State struct {
  // Matcher Function returns true on a match.
  Matcher StateMatcherFunc
}

// Returns true on matching rune
func (s *State) Match(r rune) []Matcher {
  return s.Matcher(r)
}

type Lexer struct {
  States []Matcher
  Matched string
  Hooks []Lexer
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
      var nextStates []Matcher
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
  Nexts []Matcher
}

func (rm *RuneMatcher) Match(r rune) []Matcher {
  if r == rm.MatchRune {
    return rm.Nexts
  } else {
    return []Matcher{}
  }
}

// Creates chain of RuneMatchers to match
// an entire string.
func StringMatcher(match string) (first, last *RuneMatcher) {
  for _, r := range match {
    rm := &RuneMatcher{ MatchRune: r }
    if last != nil {
      last.Nexts = []Matcher{ Matcher(rm) }
    } else {
      first = rm
    }
    last = rm
  }
  return
}

func main() {
  runes := make(chan rune)

  matcher := func(f func(rune)[]Matcher) Matcher {
    return Matcher(&State{ Matcher: f })
  }

  // Digit state
  var digit Matcher
  digit = matcher(func(r rune) []Matcher {
    if unicode.IsDigit(r) {
      return []Matcher{ digit, nil }
    } else {
      return []Matcher{}
    }
  })

  // Hex state
  var hex Matcher
  hex = matcher(func(r rune) []Matcher {
    if unicode.Is(unicode.Hex_Digit, r) {
      return []Matcher{ hex, nil }
    } else {
      return []Matcher{}
    }
  })

  // Number State
  numbf, numbl := StringMatcher("0b")
  numcf, numcl := StringMatcher("0c")
  numhf, numhl := StringMatcher("0x")

  // Identifier state
  var id Matcher
  id = matcher(func(r rune) []Matcher {
    if unicode.IsLetter(r) {
      return []Matcher{ id, nil }
    } else {
      return []Matcher{}
    }
  })

  // Whitespace state, parameterized
  whitespace := func(m ...Matcher) Matcher {
    var ws Matcher
    ws = matcher(func(r rune) []Matcher {
      if unicode.IsSpace(r) {
        var a = []Matcher{ ws }
        a = append(a, m...)
        return a
      } else {
        return []Matcher{}
      }
    })
    return ws
  }

  first, last := StringMatcher("var")
  last.Nexts = []Matcher{ whitespace(id) }

  l := Lexer{ States: []Matcher{ first } }

  // Asynchrounously lex input
  out := l.Lex(runes)

  // Asynchrounously create input and close channel.
  // On close of channel, l.Lex will close out.
  go func() {
    rdr := bufio.NewReader(os.Stdin)
    for line, _ := rdr.ReadString('\n'); len(line) > 0; {
      for _, r := range line {
        runes <- r
      }
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
