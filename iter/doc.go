// Package iter provides an alternative way to use iterators that is ergonomic
// with Go's limited form of while loop. It implements the scanner idiom
// popularized by bufio.Scanner.
//
// Example:
//
//     iter := iter.NewPairScanner(tree.NewIterator())
//     for iter.Scan() {
//             fmt.Println(iter.Result())
//     }
//
// Versus:
//
//     iter := tree.NewIterator()
//     for k, v, ok := iter.Next(); ok; k, v, ok = iter.Next() {
//             fmt.Println(k, v)
//     }
//
package iter
