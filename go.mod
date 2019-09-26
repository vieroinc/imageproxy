module willnorris.com/go/viero.tv/imageproxy

require (
	cloud.google.com/go v0.37.1
	github.com/PaulARoy/azurestoragecache v0.0.0-20170906084534-3c249a3ba788
	github.com/aws/aws-sdk-go v1.19.0
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/bradfitz/gomemcache v0.0.0-20190329173943-551aad21a668
	github.com/die-net/lrucache v0.0.0-20181227122439-19a39ef22a11
	github.com/disintegration/imaging v1.6.0
	github.com/garyburd/redigo v1.6.0
	github.com/gregjones/httpcache v0.0.0-20190212212710-3befbb6ad0cc
	github.com/jamiealquiza/envy v1.1.0
	github.com/muesli/smartcrop v0.2.1-0.20181030220600-548bbf0c0965
	github.com/peterbourgon/diskv v0.0.0-20171120014656-2973218375c3
	github.com/quirkey/magick v0.0.0-20140324185457-b37664054620
	github.com/rwcarlsen/goexif v0.0.0-20190401172101-9e8deecbddbd
	golang.org/x/image v0.0.0-20190321063152-3fc05d484e9f
	gopkg.in/gographics/imagick.v2 v2.5.0
	gopkg.in/gographics/imagick.v3 v3.2.0
	gopkg.in/h2non/bimg.v1 v1.0.19
	willnorris.com/go/gifresize v1.0.0
	willnorris.com/go/imageproxy v0.9.0
)

// temporary fix to https://github.com/golang/lint/issues/436 which still seems to be a problem
replace github.com/golang/lint => github.com/golang/lint v0.0.0-20190227174305-8f45f776aaf1

// local copy of envy package without cobra support
replace github.com/jamiealquiza/envy => ./third_party/envy

go 1.13
