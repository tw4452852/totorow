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

``` {prettyprint}
(METHOD) (URL Pattern) (Controller.Action)
```

And this is a example from the official website:

``` {prettyprint linenums}
> # conf/routes
> # This file defines all application routes (Higher priority routes first)
> GET    /login                 Application.Login      <b># A simple path</b>
> GET    /hotels/?              Hotels.Index           <b># Match /hotels and
>> /hotels/ (optional trailing slash)</b>
> GET    /hotels/{id}           Hotels.Show            <b># Extract a URI argument
>> (matching /[^/]+/)</b>
> POST   /hotels/{<[0-9]+>id}   Hotels.Save            <b># URI arg with custom
>> regex</b>
> WS     /hotels/{id}/feed      Hotels.Feed            <b># WebSockets.</b>
> POST   /hotels/{id}/{action}  Hotels.{action}        <b># Automatically route
>> some actions.</b>
> GET    /public/               staticDir:public       <b># Map /app/public
>> resources under /public/...</b>
> *      /{controller}/{action} {controller}.{action}  <b># Catch all; Automatic
> *      URL generation</b>
```
