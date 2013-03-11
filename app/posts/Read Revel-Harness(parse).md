Read Revel - Harness(parse) | 2012-12-20
# Read Revel - Harness(parse)

If you ever used the revel, you will find that there is no main program in the first time.
But once you have inputed cmd `revel run /path/to/app`, you will find there is a `main.go` in
`app/tmp` directory.
More than that, if you have amend some files and browser the page from net, it will refresh
automatically.
There must be a daemon to generate `main.go` and rebuild the whole project if some changes happened.
Yeah, this is what we should talk about - harness module.

This module do following three things:

~~~ {prettyprint}

// It has a couple responsibilities:
// 1. Parse the user program, generating a main.go file that registers
//    controller classes and starts the user's server.
// 2. Build and run the user program.  Show compile errors.
// 3. Monitor the user source and re-build / restart the program when necessary.
~~~

## Parse

Before talking about the details of the parsing source code, let's see some global path variables.

~~~ {prettyprint}

SourcePath:		$GOPATH
ImportPath: 	github.com/tw4452852/totorow
RevelPath:		$GOPATH/github.com/github.com/robfig/revel
AppName:		totorow
BasePath:		$SourcePath/$ImportPath
AppPath:		$BasePath/app
ViewsPath:		$AppPath/views
CodePaths:		$AppPath:/path/to/modules
ConfPaths:		$BasePath/conf:$RevelPath/conf
TemplatePaths:	$viewsPath:$RevelPath/templates:/path/to/modules/views
~~~

Ok, let's start the parse journey. The start point is the function `ProcessSource`. It is

~~~ {prettyprint}
func ProcessSource(roots []string) (*SourceInfo, *rev.Error)
~~~
- roots is the `CodePaths`(see above).
- the `SourceInfo` contain all the informations that found.

### Structures

~~~ {prettyprint}
// SourceInfo is the top-level struct containing all extracted information
// about the app source code, used to generate main.go.
type SourceInfo struct {
	// ControllerSpecs lists type info for all structs found under
	// app/controllers/... that embed (directly or indirectly) rev.Controller.
	ControllerSpecs []*TypeInfo
	// ValidationKeys provides a two-level lookup.  The keys are:
	// 1. The fully-qualified function name,
	//    e.g. "github.com/robfig/revel/samples/chat/app/controllers.(*Application).Action"
	// 2. Within that func's file, the line number of the (overall) expression statement.
	//    e.g. the line returned from runtime.Caller()
	// The result of the lookup the name of variable being validated.
	ValidationKeys map[string]map[int]string
	// TestSuites list the types that constitute the set of application tests.
	TestSuites []*TypeInfo
	// A list of import paths.
	// Revel notices files with an init() function and imports that package.
	InitImportPaths []string
}
~~~

The global organization of these structures are like this:

![type_org](/public/images/read_revel/type_org.png)

`TestSuites` and `controllerSpecs` are the same structures in representing.
We will take a look at `controllerSpecs` in details.
A sample `controllerSpec` is just like this:

~~~ {prettyprint}
type TypeInfo struct {
	StructName  string // e.g. "Application"
	ImportPath  string // e.g. "github.com/robfig/revel/samples/chat/app/controllers"
	PackageName string // e.g. "controllers"
	MethodSpecs []*MethodSpec

	// Used internally to identify controllers that indirectly embed *rev.Controller.
	embeddedTypes []*embeddedTypeName
}

// methodCall describes a call to c.Render(..)
// It documents the argument names used, in order to propagate them to RenderArgs.
type methodCall struct {
	Path  string // e.g. "myapp/app/controllers.(*Application).Action"
	Line  int
	Names []string
}

type MethodSpec struct {
	Name        string        // Name of the method, e.g. "Index"
	Args        []*MethodArg  // Argument descriptors
	RenderCalls []*methodCall // Descriptions of Render() invocations from this Method.
}

type MethodArg struct {
	Name       string   // Name of the argument.
	TypeExpr   TypeExpr // The name of the type, e.g. "int", "*pkg.UserType"
	ImportPath string   // If the arg is of an imported type, this is the import path.
}

// TypeExpr provides a type name that may be rewritten to use a package name.
type TypeExpr struct {
	Expr     string // The unqualified type expression, e.g. "[]*MyType"
	PkgName  string // The default package idenifier
	pkgIndex int    // The index where the package identifier should be inserted.
}

type embeddedTypeName struct {
	ImportPath, StructName string
}
~~~

![controller_specs](/public/images/read_revel/controller_specs.png)

Now, you may understand the relationship and meanings of above structure.
Next step, we will go through the procedure of building up this organization.

With the help of `go` package in standard lib, analysing go code becomes very simple.
`ProcessSource` just go through all the directories in the `CodePaths`

~~~ {prettyprint}
for _, root := range roots {
	// Start walking the directory tree.
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Println("Error scanning app source:", err)
			return nil
		}

		if !info.IsDir() || info.Name() == "tmp" {
			return nil
		}
		...
~~~
And find all the packages in the directories

~~~ {prettyprint}
var pkgs map[string]*ast.Package
fset := token.NewFileSet()
pkgs, err = parser.ParseDir(fset, path, func(f os.FileInfo) bool {
	return !f.IsDir() && !strings.HasPrefix(f.Name(), ".") && strings.HasSuffix(f.Name(), ".go")
}, 0)
~~~
Of course, the `main` package is skipped. And go through the packages that founded.

~~~ {prettyprint}
// Skip "main" packages.
delete(pkgs, "main")
...
srcInfo = appendSourceInfo(srcInfo, processPackage(fset, pkgImportPath, path, pkg))
~~~

Within every package, it goes through all the files in the package and extracts all the structures
and methods firstly.

~~~ {prettyprint}
for _, file := range pkg.Files {

	// Imports maps the package key to the full import path.
	// e.g. import "sample/app/models" => "models": "sample/app/models"
	imports := map[string]string{}

	// For each declaration in the source file...
	for _, decl := range file.Decls {

		if scanControllers {
			// Match and add both structs and methods
			addImports(imports, decl, pkgPath)
			structSpecs = appendStruct(structSpecs, pkgImportPath, pkg, decl, imports)
			appendAction(fset, methodSpecs, decl, pkgImportPath, pkg.Name, imports)
		} else if scanTests {
			addImports(imports, decl, pkgPath)
			structSpecs = appendStruct(structSpecs, pkgImportPath, pkg, decl, imports)
		}
~~~
- Note: boolean variables `scanControllers` and `scanTests` depend on whether the `pkgImportPath`
  contains the specified directory respectively.

~~~ {prettyprint}
scanControllers = strings.HasSuffix(pkgImportPath, "/controllers") ||
	strings.Contains(pkgImportPath, "/controllers/")
scanTests = strings.HasSuffix(pkgImportPath, "/tests") ||
	strings.Contains(pkgImportPath, "/tests/")
)
~~~

If a structure has anonymous fields, it will be logged in the `embeddedTypes` in the
`controllerSpecs`. It will be used for the filtering later.

~~~ {prettyprint}
for _, field := range structType.Fields.List {
	// If field.Names is set, it's not an embedded type.
	if field.Names != nil {
		continue
	}
	...
	controllerSpec.embeddedTypes = append(controllerSpec.embeddedTypes, &embeddedTypeName{
		ImportPath: importPath,
		StructName: typeName,
	})
~~~

- Only those methods that satisfy `func (receiver) FunctionName(...) rev.Result` will be
added.

After finding all the structures specs, it filter out the controller specs and test suits specs by
`findTypesThatEmbed`

~~~ {prettyprint}
// Returnall types that (directly or indirectly) embed the target type.
func findTypesThatEmbed(targetType string, specs []*TypeInfo) (filtered []*TypeInfo)
~~~
- `targetType` is "github.com/robfig/revel.Controller" for controller specs.
- "github.com/robfig/revel.TestSuite" for test suits specs.

The `findTypesThatEmbed` function is interesting, we will take a look for a while.
There is a queue to save the uncheck type names.
At first time, it only contain the input parameter `targetType`.

~~~ {prettyprint}
nodeQueue := []string{targetType}
for len(nodeQueue) > 0 {
}
return
~~~

Every loop, it get the head element from the queue as the expected type name.

~~~ {prettyprint}
		controllerSimpleName := nodeQueue[0]
		nodeQueue = nodeQueue[1:]
~~~

To find all the structures, once found a expected structure, except for appending it to the result,
it will also be added into the `nodeQueue`.
So, if type A embed the expected type, then all the structures that embed type A are also the
expected types.

~~~ {prettyprint}
for _, spec := range specs {
	if rev.ContainsString(nodeQueue, spec.String()) {
		continue // Already added
	}

	// Look through the embedded types to see if the current type is among them.
	for _, embeddedType := range spec.embeddedTypes {

		// If so, add this type's simple name to the nodeQueue, and its spec to
		// the filtered list.
		if controllerSimpleName == embeddedType.String() {
			nodeQueue = append(nodeQueue, spec.String())
			filtered = append(filtered, spec)
			break
		}
	}
}
~~~

Well, all the procedures of the parse source progress are here. The parsing strategy is

![parse_strategy](/public/images/read_revel/parse_strategy.png)

To be continue...
