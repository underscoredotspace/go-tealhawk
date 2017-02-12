angular.module('tweetApp', [])

.directive('tweets', function(tweetService) {
  return {
    restrict: 'C',
    link: function(scope, element, attribs) {
      scope.tweets = []
      tweetService(function(tweet) {
          scope.tweets.unshift(tweet)
          if (scope.tweets.length>=50) scope.tweets.splice(40, 10)
          setTimeout(function() {
            scope.$apply()
          }, 0)
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