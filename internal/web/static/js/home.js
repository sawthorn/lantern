const root = document.getElementById('homepage');
if (!root || root.dataset.initialized) {
  root.dataset.initialized = 'true';
  const $ = selector => root.querySelector(selector);

  const albumURL = id => `/albums/${encodeURIComponent(id)}`;
  const coverURL = id => `/api/covers/${encodeURIComponent(id)}`;
  const query = $('#hp-query');
  const results = $('#hp-results');
  const resultGrid = $('#hp-result-grid');
  const searchStatus = $('#hp-search-status');
  let debounceTimer;
  let controller;
  let revision = 0;

  function clearSearch() {
    query.value = '';
    updateSearch();
    query.focus();
  }

  $('#hp-clear').addEventListener('click', clearSearch);

  function makeAlbumCard(album) {
    const link = document.createElement('a');
    link.className = 'hp-album';
    link.href = albumURL(album.id);

    const cover = document.createElement('img');
    cover.className = 'hp-cover';
    cover.src = coverURL(album.id);
    cover.alt = '';
    cover.width = cover.height = 320;
    cover.loading = 'lazy';

    const title = document.createElement('h3');
    title.className = 'hp-album-title';
    title.textContent = album.title;

    const artist = document.createElement('p');
    artist.className = 'hp-album-artist';
    artist.textContent = album.artist;

    link.append(cover, title, artist);
    return link;
  }

  async function findAlbums(term, signal) {
    const params = new URLSearchParams({ q: term });

    const response = await fetch(`/api/search?${params}`, {
      signal,
      headers: {
        Accept: 'application/json'
      }
    });

    if (!response.ok) {
      throw new Error(`Search failed: HTTP ${response.status}`);
    }

    const data = await response.json();
    return data.albums;
  }

  /**
  * Updates the album search UI and schedules an API search.
  *
  * Searches are debounced so we dont send a request for every keystroke.
  * When a new search starts, the previous request is aborted and its revision
  * becomes stale. The revision check is still necessary because aborting a
  * request does not guarantee that its result cannot arrive later.
  *
  * `immediate` skips the debounce delay, which is useful for actions such as
  * pressing Enter.
  *
  * @param {boolean} immediate Whether to run the search immediately instead
  *   of waiting for the normal debounce delay.
  */
  function updateSearch(immediate = false) {
    // cancel previous requests for debounce
    clearTimeout(debounceTimer);
    controller?.abort();

    const current = ++revision;
    const term = query.value.trim();

    $('#hp-clear').hidden = query.value.length === 0;
    results.hidden = !term;
    resultGrid.replaceChildren();
    results.setAttribute('aria-busy', String(Boolean(term)));
    searchStatus.textContent = term ? 'Searching…' : '';

    if (!term) return;

    controller = new AbortController();
    const signal = controller.signal;

    debounceTimer = setTimeout(async () => {
      try {
        const matches = await findAlbums(term, signal);

        // avoid race condition between planned requests
        if (signal.aborted || current !== revision || !root.isConnected) return;

        if (
          !Array.isArray(matches) ||
          matches.some(album =>
            !album ||
            !Number.isInteger(album.id) ||
            album.id < 0 ||
            typeof album.title !== 'string' ||
            typeof album.artist !== 'string'
          )
        ) {
          throw new Error('Expected an array of album summaries');
        }

        resultGrid.replaceChildren(...matches.map(makeAlbumCard))

        searchStatus.textContent = matches.length
          ? `${matches.length} album${matches.length === 1 ? '' : 's'} found`
          : `No albums found for “${term}”. Try another title or artist.`;
      } catch (error) {
        if (signal.aborted || current !== revision || !root.isConnected) return;

        searchStatus.textContent =
          'Couldn’t search albums. Press Enter to try again.';
      } finally {
        if (current === revision) {
          results.setAttribute('aria-busy', 'false');
        }
      }
    }, immediate ? 0 : 250);
  }

  $('#hp-search').addEventListener('submit', event => {
    event.preventDefault();
    updateSearch(true);
  });

  query.addEventListener('input', () => updateSearch());

  query.addEventListener('keydown', event => {
    if (event.key === 'Escape') {
      event.preventDefault();
      clearSearch();
    }
  });
}
