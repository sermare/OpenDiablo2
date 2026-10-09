// Package d2replay records the inputs of a seeded game session and replays
// them against a simulation, hashing the simulation state after every frame.
// Two runs with the same seed and inputs must produce identical hash traces;
// any difference is nondeterminism (global math/rand, time, map iteration
// order, uuids) leaking into simulation code.
//
// The package is pure: a simulation only has to implement Sim, so it can be
// driven by tests without game data or a GUI.
package d2replay
