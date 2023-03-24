import uPlot from 'uplot';

export type UPlotData = number | null | undefined;
export type FacetedData = [
  null: null,
  series: [
    xValues: UPlotData[],
    yValues: UPlotData[],
    sizes: UPlotData[] | null,
    fills: UPlotData[] | null,
    strokes: UPlotData[] | null,
    labels: (number | string)[] | null,
  ],
];

export type UPlotAxisSplits = (
  u: uPlot,
  axisIndex: number,
  min: UPlotData,
  max: UPlotData,
) => number[];

export interface UPlotScatterProps {
  data: FacetedData;
  options: Partial<uPlot.Options>;
  tooltipLabels: (string | null)[];
}

export interface UPlotPoint {
  idx: number;
  seriesIdx: number;
}
