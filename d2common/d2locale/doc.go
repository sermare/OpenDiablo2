// Package d2locale describes the languages of the retail Diablo II installs (enUS, deDE, esES,
// frFR, itIT, jaJP, koKR, plPL, ptBR, ruRU, zhCN, zhTW) and the language dependent behaviour the
// engine needs: which string table/font directories to load, how .tbl text is decoded into the
// glyph codes used by the locale fonts, number formatting, magic item name composition and text
// wrapping. It is a pure package and never touches game files.
package d2locale
