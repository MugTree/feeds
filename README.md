## Issues

- [x] The article read action that triggers on intersect - this needs to be done on a user action. One suggestion would be to make visible a button in the lower right corner when the intersect is "crossed"
- [x] There is an error with the rendering by the looks where new notes are either not being written or a being skipped for display
      Check the template logic
      View the state coming in to check that is correct on the initial call and on updates
      Is the data being added to the margin_notes table
- [] Add an html with a direct import and a categorisation function
  I can then use it to import article on an ad hoc basis and run them through then the formatter and store directly in the caceh
  Useful for technical docs and the scrapers arent always going to work
