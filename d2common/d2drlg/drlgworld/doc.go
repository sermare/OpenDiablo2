// Package drlgworld ports the Act 1 world layout: DRLG_PlaceOutdoorLevelsInWorld
// (0x679ff0), the randomised depth-first search that decides where the Act 1
// wilderness levels, the Rogue Encampment, the Burial Grounds and the
// Monastery cluster sit relative to each other (drlg2.md B.1).
//
// Verified in the notes: the search loop, the five placement functions, the
// cluster tables and the compat table, the post-pass writes (town file index,
// level 27 exit side) and the exit flag table. UNVERIFIED (kept as noted):
// which rectangle ValidateCluster2 inflates, and whether each cluster call
// copies the DRLG seed afresh (assumed: yes, both clusters start from the same
// copy, because the real seed is only touched in pass 3).
package drlgworld
