// Adapted from code by Matt Walters https://www.mattwalters.net/posts/2018-03-28-hugo-and-lunr/

(function ($) {
  'use strict';

  $(document).ready(function () {
    const $searchInput = $('.td-search input');

    //
    // Register handler
    //

    $searchInput.on('change', (event) => {
      render($(event.target));

      // Hide keyboard on mobile browser
      $searchInput.blur();
    });

    // Prevent reloading page by enter key on sidebar search.
    $searchInput.closest('form').on('submit', () => {
      return false;
    });

    //
    // Lunr
    //

    let idx = null; // Lunr index
    const resultDetails = new Map(); // Will hold the data for the search results (titles and summaries)

    // Set up for an Ajax call to request the JSON data file that is created by Hugo's build process
    $.ajax($searchInput.data('offline-search-index-json-src')).then((data) => {
      idx = lunr(function () {
        this.ref('ref');

        // If you added more searchable fields to the search index, list them here.
        // Here you can specify searchable fields to the search index - e.g. individual toxonomies for you project
        // With "boost" you can add weighting for specific (default weighting without boost: 1)
        this.field('title', { boost: 5 });
        this.field('categories', { boost: 3 });
        this.field('tags', { boost: 3 });
        // this.field('projects', { boost: 3 }); // example for an individual toxonomy called projects
        this.field('description', { boost: 2 });
        this.field('body');

        data.forEach((doc) => {
          this.add(doc);

          resultDetails.set(doc.ref, {
            title: doc.title,
            excerpt: doc.excerpt,
          });
        });
      });

      $searchInput.trigger('change');
    });

    const render = ($targetSearchInput) => {
      //
      // Dispose existing popover
      //

      {
        let popover = bootstrap.Popover.getInstance($targetSearchInput[0]);
        if (popover !== null) {
          popover.dispose();
        }
      }

      //
      // Search
      //

      if (idx === null) {
        return;
      }

      const searchQuery = $targetSearchInput.val();
      if (searchQuery === '' || searchQuery.length < 2) {
        return;
      }

      const results = idx
        .query((q) => {
          const tokens = lunr.tokenizer(searchQuery.toLowerCase());
          tokens.forEach((token) => {
            const queryString = token.toString();
            // Exact match gets highest priority
            q.term(queryString, {
              boost: 100,
            });
            // Wildcard match for partial matches
            if (queryString.length > 2) {
              q.term(queryString, {
                wildcard:
                  lunr.Query.wildcard.LEADING | lunr.Query.wildcard.TRAILING,
                boost: 10,
              });
            }
            // Fuzzy match only for longer terms
            if (queryString.length > 4) {
              q.term(queryString, {
                editDistance: 1,
                boost: 1,
              });
            }
          });
        })
        .filter((result) => {
          // Filter out results with very low relevance scores
          return result.score > 0.5;
        })
        .slice(0, $targetSearchInput.data('offline-search-max-results'));

      //
      // Make result html
      //

      const $html = $('<div>');

      $html.append(
        $('<div>')
          .css({
            display: 'flex',
            justifyContent: 'space-between',
            marginBottom: '1em',
          })
          .append(
            $('<span>').text('Search results').css({ fontWeight: 'bold' })
          )
          .append(
            $('<span>').addClass('td-offline-search-results__close-button')
          )
      );

      const $searchResultBody = $('<div>').css({
        maxHeight: `calc(100vh - ${
          $targetSearchInput.offset().top - $(window).scrollTop() + 180
        }px)`,
        overflowY: 'auto',
      });
      $html.append($searchResultBody);

      if (results.length === 0) {
        $searchResultBody.append(
          $('<div>')
            .addClass('text-center py-4')
            .append(
              $('<p>').addClass('mb-2').html(`<strong>No results found for "${searchQuery}"</strong>`)
            )
            .append(
              $('<small>').addClass('text-muted').text('Try using different keywords or check spelling')
            )
        );
      } else {
        results.forEach((r) => {
          const doc = resultDetails.get(r.ref);
          
          // Fix: Use the ref directly as href, ensuring it starts with /
          let href = r.ref;
          if (!href.startsWith('/')) {
            href = '/' + href;
          }

          const $entry = $('<div>').addClass('mt-4');

          // Show relevance score for debugging (optional)
          const relevancePercent = Math.round(r.score * 100);
          $entry.append(
            $('<small>')
              .addClass('d-block text-body-secondary')
              .html(`${r.ref} <span class="badge bg-secondary ms-2">${relevancePercent}%</span>`)
          );

          $entry.append(
            $('<a>')
              .addClass('d-block fw-bold')
              .css({
                fontSize: '1.2rem',
                cursor: 'pointer',
              })
              .attr('href', href)
              .text(doc.title)
              .on('click', function(e) {
                // Ensure navigation works
                window.location.href = href;
              })
          );

          // Highlight matching text in excerpt
          let excerpt = doc.excerpt || '';
          const searchTerms = searchQuery.toLowerCase().split(/\s+/);
          searchTerms.forEach(term => {
            if (term.length > 2) {
              const regex = new RegExp(`(${term})`, 'gi');
              excerpt = excerpt.replace(regex, '<mark>$1</mark>');
            }
          });

          $entry.append($('<p>').html(excerpt));

          $searchResultBody.append($entry);
        });
      }

      $targetSearchInput.one('shown.bs.popover', () => {
        $('.td-offline-search-results__close-button').on('click', () => {
          $targetSearchInput.val('');
          $targetSearchInput.trigger('change');
        });
      });

      const popover = new bootstrap.Popover($targetSearchInput, {
        content: $html[0],
        html: true,
        customClass: 'td-offline-search-results',
        placement: 'bottom',
        trigger: 'manual',
      });
      popover.show();
    };
  });
})(jQuery);
