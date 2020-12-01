# Responsive
Tags: parallelizable

Specification to test the responsive design elements.

## Sign in

* Sign in as "user-w-pw" with "special-pw"
* Navigate to dashboard page

## Look for responsive elements

* Navigate to dashboard page

* Should have element "nav[class^='Navigation'] a[aria-label^='Docs']" present

* Should not have element "nav[class^='Navigation'] a[aria-label^='Overflow']" present

* Switch to mobile view

* Should not have element "nav[class^='Navigation'] a[aria-label^='Docs']" present

* Should have element "nav[class^='Navigation'] a[aria-label^='Overflow']" present

* Switch to desktop view

## Sign out

* Sign out
