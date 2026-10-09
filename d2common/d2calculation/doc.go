// Package d2calculation defines the node types of the upstream OpenDiablo2 expression tree
// for calculation strings in the game tables. It is used by d2records (item, missile and skill
// description records) together with d2lexer and d2parser. It is upstream code. The fork
// evaluates the skills.txt and missiles.txt calc columns with the separate d2calc compiler
// (d2common/d2calc), so this package is not on the path the skill pipeline takes and has not
// been compared against the original.
package d2calculation
