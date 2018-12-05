package foo

// This does not cause an error at stage-1 and stage-2 since stage-2 is skipped.
// In imoprt fixer, this path is reported 'cannot resolve'
import "/path/to/unknown/package"
