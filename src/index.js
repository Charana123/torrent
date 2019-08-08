import videojs from 'video.js'

var player = videojs('video-player', options, () => {
  this.play();
});
