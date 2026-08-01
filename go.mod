// The module path must equal the repository path, because Go derives every
// import line and the go install command from it (DEC-69). Renaming the
// repository after the first public push breaks both.
module github.com/donislawdev/qadatagenerator

// Pinned, not a minimum. Determinism is an unbreakable rule in this project,
// so the toolchain that produced a dataset is part of what has to be
// reproducible. CI pins the same version through GOTOOLCHAIN.
go 1.26.5
