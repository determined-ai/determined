export type UPlotData = number | null | undefined;
export type FacetedData = [
  null: null,
  series: [
    xValues: UPlotData[],
    yValues: UPlotData[],
    sizes: UPlotData[] | null,
    colors: UPlotData[] | null,
    labels: (number | string)[] | null,
  ],
];
