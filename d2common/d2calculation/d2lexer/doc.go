// Package d2lexer is the upstream tokenizer for calculation strings (identifiers, numbers,
// operators, parentheses). It feeds d2parser and is reachable only through d2calculation and
// d2records. It has its own unit tests; the token grammar is not compared against the original
// game's calc compiler (d2calc is the fork's port of that).
package d2lexer
