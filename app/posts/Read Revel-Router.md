# Read Revel source code - Router

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

#### Methods - NewRoute

Firstly, we go through the New-method, the method definition:

~~~ {prettyprint}
func NewRoute(method, path, action string) (r *Route)
~~~
method, path and action are corresponding to the elements in syntax respectively.

~~~ {prettyprint linenums:45}
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
- PathPattern is just the path itself and staticDir is what is following the "staticDir"

Except the static routes, remaining cases are depend on the URL.

Now, the router only support the absolute path, so:

~~~ {prettyprint linenums:71}
// TODO: Support non-absolute paths
if !strings.HasPrefix(r.Path, "/") {
	ERROR.Print("Absolute URL required.")
	return
}
~~~

Then it handle embedded arguments:

~~~ {prettyprint linenums:78}
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

~~~ {prettyprint linenums:85}
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
All the args in the URL are collected in r.args, a slice of *arg:

~~~ {prettyprint}
type arg struct {
	name       string
	index      int
	constraint *regexp.Regexp
}
~~~
- name: is the var group in argsPattern
- constraint: is the pattern group in argsPattern

The next step is to generate pathPattern, due to the above work, it just group name according to the
var name in url regexp

~~~ {prettyprint linenums:95}
// Now assemble the entire path regex, including the embedded parameters.
// e.g. /app/{<[^/]+>id} => /app/(?P<id>[^/]+)
pathPatternStr := argsPattern.ReplaceAllStringFunc(normPath, func(m string) string {
	var argMatches []string = argsPattern.FindStringSubmatch(m)
	return "(?P<" + argMatches[2] + ">" + argMatches[1] + ")"
})
r.pathPattern = regexp.MustCompile(pathPatternStr + "$")
~~~

The last step is to generate actionPattern. It just used the generated args to do replacement:
`{controller} => {(?P<controller>[^/]+)}`

~~~ {prettyprint linenums:103}
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
e.g. The route record is:`GET /{controller}/{<[a-z]+>action} {controller}.{methord}`, and the generated args
slice is:

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
express the result with a *RouteMatch, its definition:

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
















































































































































































































































































































































































































































































































































































































































