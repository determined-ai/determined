export {};

// import React, { useEffect, useRef, useState } from 'react';
// import uPlot, { Options } from 'uplot';
//
// import useResize from 'hooks/useResize';
//
// import css from './ScatterPlot.module.scss';
//
// interface Props {
//   height?: number;
//   title?: string;
//   values: number[];
//   x: number[];
//   xLabel?: string;
//   y: number[];
//   yLabel?: string;
// }
//
// const CHART_HEIGHT = 200;
// const UPLOT_OPTIONS = {
//   axes: [
//     {
//       grid: { width: 1 },
//       scale: 'x',
//       side: 2,
//     },
//     {
//       grid: { width: 1 },
//       scale: 'y',
//       side: 3,
//     },
//   ],
//   cursor: {
//     points: {
//       fill: 'transparent',
//       size: 10,
//       stroke: 'var(--theme-colors-action-normal)',
//       width: 2,
//     },
//     x: false,
//     y: false,
//   },
//   legend: { show: false },
//   scales: {
//     x: { auto: true, time: false },
//     y: { auto: true, time: false },
//   },
//   series: [
//     { label: 'x' },
//     {
//       label: 'scatter',
//       points: {
//         fill: 'blue',
//         size: 4,
//         space: 0,
//         width: 0,
//       },
//       scale: 'y',
//       width: 1 / window.devicePixelRatio,
//     },
//   ],
// };
//
// const ScatterPlotUPlot: React.FC<Props> = ({
//   height = CHART_HEIGHT,
//   title,
//   values,
//   x,
//   xLabel,
//   y,
//   yLabel,
// }: Props) => {
//   const chartRef = useRef<HTMLDivElement>(null);
//   const resize = useResize(chartRef);
//   const [ chart, setChart ] = useState<uPlot>();
//
//   useEffect(() => {
//     if (!chartRef.current) return;
//
//     const options = uPlot.assign({}, UPLOT_OPTIONS, {
//       height,
//       series: [
//         { label: 'x' },
//         {
//           label: 'scatter',
//           points: {
//             fill: 'blue',
//             size: 4,
//             space: 0,
//             width: 0,
//           },
//           scale: 'y',
//           width: 1 / window.devicePixelRatio,
//         },
//       ],
//       width: chartRef.current.offsetWidth,
//     }) as Options;
//
//     if (title) options.title = title;
//     if (xLabel) (options.axes || [])[0].label = xLabel;
//     if (yLabel) (options.axes || [])[1].label = yLabel;
//
//     const plotChart = new uPlot(options, [ x, y ], chartRef.current);
//     setChart(plotChart);
//
//     return () => {
//       setChart(undefined);
//       plotChart.destroy();
//     };
//   }, [ height, title, values, x, xLabel, y, yLabel ]);
//
//   // Resize the chart when resize events happen.
//   useEffect(() => {
//     if (chart) chart.setSize({ height, width: resize.width });
//   }, [ chart, height, resize ]);
//
//   return (
//     <div className={css.base}>
//       <div ref={chartRef} />
//     </div>
//   );
// };
//
// export default ScatterPlotUPlot;
