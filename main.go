package main

import "fmt"

// Match-Nexter interface
// Allows for magic to happen inside the type.
type MatchNexter interface {
  // Is this rune a match?
  Match(rune) bool
  // Returns next possible states
  Next() []MatchNexter
}

// Matching function used by State type
type StateMatcherFunc func(rune) bool

// State type implements MatchNexter
type State struct {
  // Matcher Function returns true on a match.
  Matcher StateMatcherFunc
  // Possible next states
  // if this state is true.
  Nexts []MatchNexter
}

// Returns true on matching rune
func (s *State) Match(r rune) bool {
  return s.Matcher(r)
}

// Returns possible next states
func (s *State) Next() []MatchNexter {
  return s.Nexts
}

type Lexer struct {
  Runes <-chan rune
  States []MatchNexter
}

// Update later
type Token rune

// Uses an array of possible MatchNexters
// Each MatchNexters' Match function is called
// exactly once per set of MatchNexters.
// If Match returns true, then Next is called.
func (l *Lexer) Lex(out chan Token) {
  defer close(out)
  for r := range l.Runes {
    var nextStates []MatchNexter
    for _, s := range l.States {
      if s.Match(r) {
        out <- Token(r)
        nextStates = append(nextStates, s.Next()...)
      }
    }
    l.States = nextStates
    if len(l.States) == 0 {
      return // No more possible states.
    }
  }
}

func main() {
  runes := make(chan rune)
  out := make(chan Token)
  l := Lexer{
    Runes: runes,
    States: []MatchNexter{
      MatchNexter(&State{
        Matcher: func(r rune) bool {
          return r == 'a'
        },
        Nexts: []MatchNexter{
          MatchNexter(&State{
            Matcher: func (r rune) bool {
              return r == 'b'
            },
          }),
        },
      }),
    },
  }
  // Asynchrounously lex input
  go l.Lex(out)
  // Asynchrounously create input and close channel.
  // On close of channel, l.Lex will close out.
  go func() {
    runes <- 'a'
    runes <- 'b'
    close(runes)
  }()
  // Keep-alive until out is closed.
  for t := range out {
    fmt.Println("Found match '"+string(rune(t))+"'")
  }
}
