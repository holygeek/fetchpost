## About

fetchpost - Save HN post and comments as mails in maildir format

## HOW TO USE

1. Install:
   ```sh
   $ go get github.com/holygeek/fetchpost
   ```
2. Fetch the posts as mails:
   ````sh
   $ fetchpost 'https://news.ycombinator.com/item?id=10041477'
   ...
   Ask_HN_Any_movies_that_changed_your_life_
   ```
3. Read it with, say, mutt:
   ```sh
   $ cat <<EOF >muttrc
   set sort = threads
   ignore subject to date
   unignore x-date
   EOF

   $ mutt -F muttrc -f Ask_HN_Any_movies_that_changed_your_life_
   ```
4. To fetch new posts:
   ```sh
   $ fetchpost Ask_HN_Any_movies_that_changed_your_life_
   ```

## BUGS

Definitely!