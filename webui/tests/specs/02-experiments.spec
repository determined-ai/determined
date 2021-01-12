# Experiment List
Tags: parallelizable

Specification to test the experiment list page.

## Sign in

* Sign in as "user-w-pw" with "special-pw"
* Navigate to experiment list page

## Experiment batch operations

* Toggle all table row selection
* Table batch should have following buttons

  |table batch buttons|disabled|
  |-------------------|--------|
  |View in TensorBoard|false   |
  |Activate           |false   |
  |Pause              |true    |
  |Archive            |false   |
  |Unarchive          |true    |
  |Cancel             |false   |
  |Kill               |false   |
* Toggle all table row selection

## Filter experiments by archived

* Toggle show archived button
* Should have "4" table rows
* Toggle show archived button
* Should have "3" table rows

## Sign out

* Sign out
