# Custom Cells

This directory contains the custom cell renderers used with the Glide Table. They are adapted from the `glide-data-grid-cells` package. We need to fork them to get the styling that we want, and what we actually need from the package is a small enough portion that it is easier to maintain this way (e.g. over half the renderer code in the package is dedicated to providing editors for the cells- which we do not use).

- [avatar](./avatar.tsx) shows user avatars. is adapted for styling purposes.
- [links](./links.tsx) shows links. is adapted for styling purposes.
- [progress](./progress.tsx) shows a progress bar. is adapted to allow for passing different colors in props, and also has a shadow.
- [sparkline](./sparkline.tsx) shows graphs within the table row
- [spinner](./spinner.tsx) shows a spinner for a cell that is loading
- [tags](./tags.tsx) shows tags for experiments. is adapted for styling purposes.

In order to make your own cell, you need to create a render that satisfies the `CustomRenderer` interface. The main thing is to provide a `draw` function that takes `args` (which includes the canvas context), and `cell : GridCell` definition returned by `getCellContent`. For custom cells, you can pass whichever "props" you like through `cell.data` to be accessed in the draw function.
