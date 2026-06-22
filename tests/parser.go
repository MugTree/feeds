package main

// import (
// 	"fmt"
// 	"strings"
// )

// func main() {

// 	// replace newlines, tabs and spaces...
// 	// ------------------------------------------
// 	re := strings.NewReplacer("\n", "", "\t", "")

// 	str := `<!DOCTYPE html>
// 	<html lang="en">
// 		<head>
// 			<meta charset="UTF-8"/>
// 			<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
// 			<title>Document</title>
// 		</head>
// 		<body>
// 			<h1>Hey!</h1>
// 			<p>blah blah</p>
// 			<ul>
// 				<li>a</li>
// 				<li>b</li>
// 				<li>c</li>
// 			</ul>
// 		</body>
// 	</html>`

// 	ns := re.Replace(str)

// 	// pull thsese from the db
// 	an := []Annotation{{Start: 9, End: 11, ID: "1"}}

// 	// Hey
// 	html, err := applyAnnotations(ns, an)
// 	if err != nil {
// 		fmt.Println(err)
// 	}

// 	fmt.Println(html)
// }
