Read Revel - Router | 2012-12-05
# Read Revel - Router

This blog is based on the [Revel web framework][revel_github], So I will
introduce this framework firstly.If you are the newbie, here is the
[tutorial][revel_tuto] for you.Today, as the first part of this serial, I will
introduce the router of it.

[revel_github]:https://github.com/robfig/revel
[revel_tuto]:http://rofig.github.com/revel/tutorial/index.html

## Introduce
As a web framework, the router is necessary. It handles all kinds of URLs and
determines what to do depend on the received url.

## Syntax
The revel routing syntax is simple:

~~~ {prettyprint}
(METHOD) (URL Pattern) (Controller.Action)
~~~

And this is a example route configure file from the official website:

~~~ {prettyprint lang-bsh}
# conf/routes
# This file defines all application routes (Higher priority routes first)
GET    /login                 Application.Login      # A simple path
GET    /hotels/?              Hotels.Index           # Match /hotels and /hotels/
GET    /hotels/{id}           Hotels.Show            # Extract a URI argument (matching /[^/]+/)
POST   /hotels/{<[0-9]+>id}   Hotels.Save            # URI arg with custom regexp
WS     /hotels/{id}/feed      Hotels.Feed            # WebSockets.
POST   /hotels/{id}/{action}  Hotels.{action}        # Automatically route some actions.
GET    /public/               staticDir:public       # Map /app/public resources under /public/...
*      /{controller}/{action} {controller}.{action}  # Catch all; Automatic URL generation
~~~

## Code
Now, let's go through the source code that implement routing. All the code about routing are in
router.go.

### Data structure - Route
To express a route, it uses this structure named "Route":

~~~ {prettyprint}
type Route struct {
	Method string // e.g. GET
	Path   string // e.g. /app/{id}
	Action string // e.g. Application.ShowApp

	pathPattern   *regexp.Regexp // for matching the url path
	staticDir     string         // e.g. "public" from action "staticDir:public"
	args          []*arg         // e.g. {id} from path /app/{id}
	actionPattern *regexp.Regexp
}
~~~

As the comments say, some fields are easy to understand. If you are still a little misleading, don't
worry, it will become clear while we go on later.

#### Method - NewRoute

Firstly, we go through the New-method, the method definition:

~~~ {prettyprint}
func NewRoute(method, path, action string) (r *Route)
~~~
method, path and action are corresponding to the elements in syntax respectively.

~~~ {prettyprint}
r = &Route{
	Method: strings.ToUpper(method),
	Path:   path,
	Action: action,
}

// Handle static routes
if strings.HasPrefix(r.Action, "staticDir:") {
	if r.Method != "*" && r.Method != "GET" {
		WARN.Print("Static route only supports GET")
		return
	}

	if !strings.HasSuffix(r.Path, "/") {
		WARN.Printf("The path for staticDir must end with / (%s)", r.Path)
		r.Path = r.Path + "/"
	}

	r.pathPattern = regexp.MustCompile("^" + r.Path + "(.*)$")
	r.staticDir = r.Action[len("staticDir:"):]
	// TODO: staticFile:
	return
}
~~~
static routes case:

- It only support GET method, if not it return with a warning directly.
- Auto add trailing slash.
- `PathPattern` is just the path itself and staticDir is what is following the "staticDir"

Except the static routes, remaining cases are depend on the URL.

Now, the router only support the absolute path, so:

~~~ {prettyprint}
// TODO: Support non-absolute paths
if !strings.HasPrefix(r.Path, "/") {
	ERROR.Print("Absolute URL required.")
	return
}
~~~

Then it handle embedded arguments:

~~~ {prettyprint}
// Convert path arguments with unspecified regexes to standard form.
// e.g. "/customer/{id}" => "/customer/{<[^/]+>id}
normPath := nakedPathParamRegex.ReplaceAllStringFunc(r.Path, func(m string) string {
	var argMatches []string = nakedPathParamRegex.FindStringSubmatch(m)
	return "{<[^/]+>" + argMatches[1] + "}"
})
~~~
- nakedPathParamRegex is:

~~~ {prettyprint}
nakedPathParamRegex = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z_0-9]*)\}`)
~~~
As the comments say, it change "{id}" => "{<[^/]+>id}". Because of regexp package in go is very
powerful, the work becomes easy. [This][regexp-tuto] is a go regexp package tutorial.

[regexp-tuto]:https://github.com/StefanSchroeder/Golang-Regex-Tutorial

After above translation, all the args pattern is uniform, like this `{<arg_pattern>arg_name}`,
collect them all.

~~~ {prettyprint}
// Go through the arguments
r.args = make([]*arg, 0, 3)
for i, m := range argsPattern.FindAllStringSubmatch(normPath, -1) {
	r.args = append(r.args, &arg{
		name:       string(m[2]),
		index:      i,
		constraint: regexp.MustCompile(string(m[1])),
	})
}
~~~
argsPattern is:

~~~ {prettyprint}
argsPattern = regexp.MustCompile(`\{<(?P<pattern>[^>]+)>(?P<var>[a-zA-Z_0-9]+)\}`)
~~~
All the arguments in the URL are collected in `r.args`, a slice of *arg:

~~~ {prettyprint}
type arg struct {
	name       string
	index      int
	constraint *regexp.Regexp
}
~~~
- name: is the var group in `argsPattern`
- constraint: is the pattern group in `argsPattern`

The next step is to generate `pathPattern`, due to the above work, it just group name according to the
var name in url regexp

~~~ {prettyprint}
// Now assemble the entire path regex, including the embedded parameters.
// e.g. /app/{<[^/]+>id} => /app/(?P<id>[^/]+)
pathPatternStr := argsPattern.ReplaceAllStringFunc(normPath, func(m string) string {
	var argMatches []string = argsPattern.FindStringSubmatch(m)
	return "(?P<" + argMatches[2] + ">" + argMatches[1] + ")"
})
r.pathPattern = regexp.MustCompile(pathPatternStr + "$")
~~~

The last step is to generate `actionPattern`. It just used the generated args to do replacement:
`{controller} => {(?P<controller>[^/]+)}`

~~~ {prettyprint}
// Handle action
var actionPatternStr string = strings.Replace(r.Action, ".", `\.`, -1)
for _, arg := range r.args {
	var argName string = "{" + arg.name + "}"
	if argIndex := strings.Index(actionPatternStr, argName); argIndex != -1 {
		actionPatternStr = strings.Replace(actionPatternStr, argName,
			"(?P<"+arg.name+">"+arg.constraint.String()+")", -1)
	}
}
r.actionPattern = regexp.MustCompile(actionPatternStr)
~~~

When all the works above is done, a route is generated.Let's see a little complicate example to walk
through the entire flow.
e.g. The route record is:`GET /{controller}/{<[a-z]+>action} {controller}.{methord}`, and the
generated arguments slice is:

~~~ {prettyprint}
r.args = [
	{"controller", 0, regexp.MustCompile("[^/]+")},
	{"action", 1, "regexp.MustCompile("[a-z]+")},
]
~~~
pathPattern is:

~~~ {prettyprint}
r.pathPattern = regexp.MustCompile("/(?P<controller>[^/]+)/(?P<action>[a-z]+)$")
~~~

r.actionPattern is:

~~~ {prettyprint}
r.actionPattern = regexp.MustCompile("{(?P<controller>[^/]+)}\.{(?P<action>[a-z]+)}")
~~~


#### Method - Match
The route has the Match method that can judge whether a URL request is match this route or not. It
express the result with a `*RouteMatch`, its definition:

~~~ {prettyprint}
type RouteMatch struct {
	Action         string            // e.g. Application.ShowApp
	ControllerName string            // e.g. Application
	MethodName     string            // e.g. ShowApp
	Params         map[string]string // e.g. {id: 123}
	StaticFilename string
}
~~~

Method definition:

~~~ {prettyprint}
func (r *Route) Match(method string, reqPath string) *RouteMatch
~~~

Firstly, it check method, and it only accept HEAD and GET method.

~~~ {prettyprint}
// Check the Method
if r.Method != "*" && method != r.Method && !(method == "HEAD" && r.Method == "GET") {
	return nil
}
~~~

Then check the request URL to find arguments if any.

~~~ {prettyprint}
// Check the Path
var matches []string = r.pathPattern.FindStringSubmatch(reqPath)
if matches == nil {
	return nil
}
~~~

As the `NewRoute` method, it also check if it is a staticDir file request at first.

~~~ {prettyprint}
// If it's a static file request..
if r.staticDir != "" {
	// Check if it is specifying a module.. if so, look there instead.
	// This is a tenative syntax: "staticDir:moduleName:(directory)"
	var basePath, dirName string
	if i := strings.Index(r.staticDir, ":"); i != -1 {
		moduleName, dirName := r.staticDir[:i], r.staticDir[i+1:]
		for _, module := range Modules {
			if module.Name == moduleName {
				basePath = path.Join(module.Path, dirName)
			}
		}
		if basePath == "" {
			ERROR.Print("No such module found: ", moduleName)
			basePath = BasePath
		}
	} else {
		basePath, dirName = BasePath, r.staticDir
	}
	return &RouteMatch{
		StaticFilename: path.Join(basePath, dirName, matches[1]),
	}
}
~~~
- If `r.staticDir` contains modules(we will talk it at the following chapter of this serial), get
  `basePath` and `dirName` from the module
- Otherwise, `basePath` is from the global var `BasePath` and `dirName` is same as the `r.staticDir`.

Following is the regular URL case. Get the parameters from the previous match slice.

e.g if the route configure record is:`GET /{controller}/{method} {controller}.{method}` and the
request URL is `/tw/name`, then the parameters here is `{"controller":"tw", "method":"name",}`

~~~ {prettyprint}
// Figure out the Param names.
params := make(map[string]string)
for i, m := range matches[1:] {
	params[r.pathPattern.SubexpNames()[i+1]] = m
}
~~~

Get action, here it just find whether there is a "{" in `r.Action`. If so, replace it with the actual
value, continue with the previous example:`{controller}.{method} => tw.name`

~~~ {prettyprint}
// If the action is variablized, replace into it with the captured args.
action := r.Action
if strings.Contains(action, "{") {
	for key, value := range params {
		action = strings.Replace(action, "{"+key+"}", value, -1)
	}
}
~~~

One special case is the "404" action, In that case, return "404" action directly.

~~~ {prettyprint}
// Special handling for explicit 404's.
if action == "404" {
	return &RouteMatch{
		Action: "404",
	}
}
~~~

So far, all the things are well prepared, just spilt the action string with "." to extract the
controller and method strings.

~~~ {prettyprint}
// Split the action into controller and method
actionSplit := strings.Split(action, ".")
if len(actionSplit) != 2 {
	ERROR.Printf("Failed to split action: %s (matching route: %s)", action, r.Action)
	return nil
}

return &RouteMatch{
	Action:         action,
	ControllerName: actionSplit[0],
	MethodName:     actionSplit[1],
	Params:         params,
}
~~~

### Data structure - Router

To form a route database, revel use the structure "Router" to express it.

~~~ {prettyprint}
type Router struct {
	Routes []*Route
	path   string
}
~~~
Just a route slice and a local file path to save the database in the local storage as a file.

#### method - NewRouter
When create a router database, it just need the local file path.

~~~ {prettyprint}
func NewRouter(routesPath string) *Router {
	return &Router{
		path: routesPath,
	}
}
~~~

#### Method - Route
To find a http request in the database, just walk through the slice, if there is a route match it,
return this result with a `RouteMatch` structure, otherwise return nil.

~~~ {prettyprint}
func (router *Router) Route(req *http.Request) *RouteMatch {
	for _, route := range router.Routes {
		if m := route.Match(req.Method, req.URL.Path); m != nil {
			return m
		}
	}
	return nil
}
~~~

#### Method - Refresh
To recovery the route database from the local files, method Refresh will accomplish this work.

~~~ {prettyprint}
// Refresh re-reads the routes file and re-calculates the routing table.
// Returns an error if a specified action could not be found.
func (router *Router) Refresh() *Error {
	// Get the routes file content.
	contentBytes, err := ioutil.ReadFile(router.path)
	if err != nil {
		return &Error{
			Title:       "Failed to load routes file",
			Description: err.Error(),
		}
	}

	return router.parse(string(contentBytes), true)
}
~~~
If there is a error happened during reading the file, return a `revel.Error`. The main part is
located in a internal method `Router.parse`

~~~ {prettyprint}
func (router *Router) parse(content string, validate bool) *Error
~~~

The same as a usually way, parse the file content line by line and collect all the found route in
the `router.Routes` slice. If we are required to validate the founded route, `router.validate` will
check it.

~~~ {prettyprint}
routes := make([]*Route, 0, 10)

// For each line..
for n, line := range strings.Split(content, "\n") {
	line = strings.TrimSpace(line)
	if len(line) == 0 || line[0] == '#' {
		continue
	}

	method, path, action, found := parseRouteLine(line)
	if !found {
		continue
	}

	route := NewRoute(method, path, action)
	routes = append(routes, route)

	if validate {
		if err := router.validate(route); err != nil {
			err.Path = router.path
			err.Line = n + 1
			err.SourceLines = strings.Split(content, "\n")
			return err
		}
	}
}

router.Routes = routes
return nil
~~~

`parseRouteLine` function is to extract the method, path, action from this line.

~~~ {prettyprint linenums}
func parseRouteLine(line string) (method, path, action string, found bool) {
	var matches []string = routePattern.FindStringSubmatch(line)
	if matches == nil {
		return
	}
	method, path, action = matches[1], matches[4], matches[5]
	found = true
	return
}
~~~
The `routePattern` is

~~~ {prettyprint}
// Groups:
// 1: method
// 4: path
// 5: action
var routePattern *regexp.Regexp = regexp.MustCompile(
	"(?i)^(GET|POST|PUT|DELETE|OPTIONS|HEAD|WS|\\*)" +
		"[(]?([^)]*)(\\))?[ \t]+" +
		"(.*/[^ \t]*)[ \t]+([^ \t(]+)(.+)?([ \t]*)$")
~~~
Let me analysis it:

~~~ {prettyprint}
1:method: (?i)^(GET|POST|PUT|DELETE|OPTIONS|HEAD|WS|\\*) //case insensitivity
2: [^)]*
3: \\)
4:path: .*/[^ \t]*
5:method: [^ \t(]+
6: .+
7: [ \t]*
~~~

validate is just validate the controller and method.static routes, variable routes and 404 cases are
ignored.

~~~ {prettyprint}
// Skip static routes
if route.staticDir != "" {
	return nil
}

// Skip variable routes.
if strings.ContainsAny(route.Action, "{}") {
	return nil
}

// Skip 404s
if route.Action == "404" {
	return nil
}
~~~

Then find the controller and method from the `route.Action` and look up them, if not found, return
`revel.Error`

~~~ {prettyprint}
// We should be able to load the action.
parts := strings.Split(route.Action, ".")
if len(parts) != 2 {
	return &Error{
		Title: "Route validation error",
		Description: fmt.Sprintf("Expected two parts (Controller.Action), but got %d: %s",
			len(parts), route.Action),
	}
}

ct := LookupControllerType(parts[0])
if ct == nil {
	return &Error{
		Title:       "Route validation error",
		Description: "Unrecognized controller: " + parts[0],
	}
}

mt := ct.Method(parts[1])
if mt == nil {
	return &Error{
		Title:       "Route validation error",
		Description: "Unrecognized method: " + parts[1],
	}
}
~~~

#### Method - Reverse

The router provide a method name "Reverse", as the name says, it reverse a action string to a route
record. But it uses another structure to express it:

~~~ {prettyprint}
type ActionDefinition struct {
	Host, Method, Url, Action string
	Star                      bool
	Args                      map[string]string
}
~~~
And this structure also satisfy stringer interface:

~~~ {prettyprint}
func (a *ActionDefinition) String() string {
	return a.Url
}
~~~
We will encounter this structure later on.

Well, let's look through the Reverse method. All the method is located in a loop through the route
database. Once find the result, return it directly.

~~~ {prettyprint}
NEXT_ROUTE:
// Loop through the routes.
for _, route := range router.Routes {

	...

	return &ActionDefinition{
		Url:    url,
		Method: method,
		Star:   star,
		Action: action,
		Args:   argValues,
		Host:   "TODO",
	}
}
ERROR.Println("Failed to find reverse route:", action, argValues)
return nil
~~~

And the detail of find method is to construct two maps and compare them. So at first, it construct
the map in the target action string.

~~~ {prettyprint}
var matches []string = route.actionPattern.FindStringSubmatch(action)
if len(matches) == 0 {
	continue
}

for i, match := range matches[1:] {
	argValues[route.actionPattern.SubexpNames()[i+1]] = match
}
~~~

And the database's map:

~~~ {prettyprint}
// Create a lookup for the route args.
routeArgs := make(map[string]*arg)
for _, arg := range route.args {
	routeArgs[arg.name] = arg
}
~~~

Compare them:

~~~ {prettyprint}
// Enforce the constraints on the arg values.
for argKey, argValue := range argValues {
	arg, ok := routeArgs[argKey]
	if ok && !arg.constraint.MatchString(argValue) {
		continue NEXT_ROUTE
	}
}
~~~

If found one, generate the URL, most of work is to generate the query part.

~~~ {prettyprint}
var queryValues url.Values = make(url.Values)
path := route.Path
for argKey, argValue := range argValues {
	if _, ok := routeArgs[argKey]; ok {
		// If this arg goes into the path, put it in.
		path = regexp.MustCompile(`\{(<[^>]+>)?`+regexp.QuoteMeta(argKey)+`\}`).
			ReplaceAllString(path, url.QueryEscape(string(argValue)))
	} else {
		// Else, add it to the query string.
		queryValues.Set(argKey, argValue)
	}
}
~~~
If found in the route args, replace it with the actual value, otherwise, set a new one.

At last, connect the query part with the path.

~~~ {prettyprint}
// Calculate the final URL and Method
url := path
if len(queryValues) > 0 {
	url += "?" + queryValues.Encode()
}
~~~

And extract the method part(special case "*" method):

~~~ {prettyprint}
method := route.Method
star := false
if route.Method == "*" {
	method = "GET"
	star = true
}
~~~

FIN.
