const playBtn = document.getElementById('album-play-button')
const firstTrack = document.querySelector('.tracklist-entry')
playBtn?.addEventListener('click', () => {
  firstTrack?.click()
})

document.addEventListener('error', (event) => {
  if (event.target instanceof HTMLImageElement && event.target.matches('[data-album-cover]')) {
    event.target.hidden = true;
  }
}, true);

document.querySelectorAll('[data-album-cover]').forEach((image) => {
  if (image.complete && image.naturalWidth === 0) image.hidden = true;
});
