// Package d2parser is the upstream parser that turns a calculation string into a
// d2calculation node tree using d2lexer. It is used by d2records for the older, pre-pipeline
// code paths. It has unit tests but is an approximation of the original calc language; the
// implementation checked against the binary notes is d2common/d2calc.
package d2parser
