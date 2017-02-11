angular.module('tweetApp', [])

.directive('tweets', function(tweetService) {
  return {
    restrict: 'C',
    link: function(scope, element, attribs) {
      tweetService(function(tweet) {
        console.log(tweet.text)
      })
    }
  }
})

.service('tweetService', function() {
  var socket = io.connect()
  var tweetHandle = null

  return function(watch) {
    tweetHandle = watch
    socket.on('connect', function(){
      console.log('connected')
    })
    socket.on('tweet', function(tweet) {
      if (tweetHandle) tweetHandle(JSON.parse(tweet))
    })
  }
})