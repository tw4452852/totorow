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
The revel routing syntax is very easy:


```prettyprint
(METHOD) (URL Pattern) (Controller.Action)
```
