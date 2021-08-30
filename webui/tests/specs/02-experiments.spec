# Experiment List
Tags: parallelizable

Specification to test the experiment list page.

## Experiment batch operations

* Navigate to experiment list page
* Toggle all table row selection
* Table batch should have following buttons

  |table batch buttons|disabled|
  |-------------------|--------|
  |View in TensorBoard|false   |
  |Activate           |false   |
  |Pause              |true    |
  |Archive            |false   |
  |Unarchive          |false   |
  |Cancel             |false   |
  |Kill               |false   |

// ## Filter experiments by archived

// * Scroll table to the "right"
// * Filter table header "Archived" with option "Archived"
// * Should have "1" table rows
// * Filter table header "Archived" with option "Unarchived"
// * Should have "3" table rows
