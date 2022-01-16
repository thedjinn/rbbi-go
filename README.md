# RBBI for Go

This is a Go port of ICU4C's Rule-Based Break Iterator (RBBI), an algorithm
for extracting various types of breaks (character/grapheme cluster, line,
sentence, and word) from unicode strings. It supports both forward and reverse
iteration.

[![Tests](https://img.shields.io/github/workflow/status/thedjinn/rbbi-go/tests)](https://github.com/thedjinn/rbbi-go/actions/workflows/tests.yml)
[![Apache License](https://img.shields.io/github/license/thedjinn/rbbi-go?color=blue)](https://github.com/thedjinn/rbbi-go/blob/main/LICENSE)
[![Documentation](https://pkg.go.dev/badge/github.com/thedjinn/rbbi-go)](https://pkg.go.dev/github.com/thedjinn/rbbi-go)

## How to use

Using the break iterater consists of two easy steps:

1. Instantiate a RBBI instance using one of the four constructors, e.g.
   `NewCharacterRBBI()` for character break detection.

2. Provide the break iterator with a struct implementing the Cursor interface.
   For iteration over simple strings the StringCursor struct can be used, and
   for more complex backing stores the interface can be implemented using e.g.
   a wrapper struct.

Of course, a simple code example is worth a thousand words:

    str := "the string to iterate over"
    cursor := rbbi.NewStringCursor(str)

    iter := rbbi.NewCharacterRBBI()
    iter.SetCursor(cursor)

    for {
        position, ok := iter.Next()

        if !ok {
            break
        }

        fmt.Println("Found a break at offset %v", position)
    }

For more information, please refer to the
[documentation](https://pkg.go.dev/github.com/thedjinn/rbbi-go).

## License

Copyright 2022 Emil Loer

Licensed under the Apache License, Version 2.0 (the "License"); you may not
use this file except in compliance with the License.  You may obtain a copy of
the License at
[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0).

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the
License for the specific language governing permissions and limitations under
the License.

This project is a derivative work based on data files and algorithms from
ICU4C, for which the following additional licensing terms apply:

Copyright © 1991-2022 Unicode, Inc. All rights reserved.

Distributed under the Terms of Use in
[https://www.unicode.org/copyright.html](https://www.unicode.org/copyright.html).

For the full license text of this entire project and its third party
dependencies, please refer to the LICENSE file in the project repository.
