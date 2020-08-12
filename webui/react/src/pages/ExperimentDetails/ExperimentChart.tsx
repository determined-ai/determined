import { PlotData } from 'plotly.js/lib/core';
import React, { useMemo } from 'react';

import MetricChart from 'components/MetricChart';
import { ValidationHistory } from 'types';

interface Props {
  id?: string;
  startTime?: string;
  validationMetric?: string;
  validationHistory?: ValidationHistory[];
}

const ExperimentChart: React.FC<Props> = ({ validationMetric, ...props }: Props) => {
  const titleDetail = validationMetric ? ` (${validationMetric})` : '';
  const title = `Best Validation Metric${titleDetail}`;

  const data: Partial<PlotData>[] = useMemo(() => {
    if (!props.startTime || !props.validationHistory) return [];

    const startTime = new Date(props.startTime).getTime();
    const textData: string[] = [];
    const xData: number[] = [];
    const yData: number[] = [];

    props.validationHistory.forEach(validation => {
      const endTime = new Date(validation.endTime).getTime();
      const x = (endTime - startTime) / 1000;
      const y = validation.validationError;
      const text = [
        `Trial ${validation.trialId}`,
        `Elapsed Time: ${x} sec`,
        `Metric Value: ${y}`,
      ].join('<br>');
      if (text && x && y) {
        textData.push(text);
        xData.push(x);
        yData.push(y);
      }
    });

    return [ {
      hovermode: 'y unified',
      hovertemplate: '%{text}<extra></extra>',
      mode: 'lines+markers',
      text: textData,
      type: 'scatter',
      x: xData,
      y: yData,
    } ];
  }, [ props.startTime, props.validationHistory ]);

  const renderPlot = useCallback(async (
    elementId: string,
    plotData: Partial<PlotData>[],
    plotLayout?: Partial<Plotly.Layout>,
    plotConfig?: Partial<Plotly.Config>,
  ) => {
    const layout = plotLayout || defaultLayout;
    const config = plotConfig || defaultConfig;
    const args: PlotArguments = [ elementId, plotData, layout, config ];

    if (isRendered) {
      await Plotly.react.apply(null, args);
    } else {
      const chart: PlotlyHTMLElement = await Plotly.newPlot.apply(null, args);
      chart.on('plotly_relayout', (event: PlotRelayoutEvent) => {
        /*
         * Preserve the zoom and pan range. When new data comes in
         * the re-rendering of the plot will render the same zoom level.
         */
        setRange({
          xaxis: [ event['xaxis.range[0]'], event['xaxis.range[1]'] ],
          yaxis: [ event['yaxis.range[0]'], event['yaxis.range[1]'] ],
        });
      });
      setIsRendered(true);
    }
  }, [ isRendered ]);

  useEffect(() => {
    const layout = clone(defaultLayout);
    layout.yaxis.type = scale;

    if (range.xaxis[0] != null && range.xaxis[1] != null &&
        range.yaxis[0] != null && range.yaxis[1] != null) {
      layout.xaxis.range = range.xaxis;
      layout.yaxis.range = range.yaxis;
    }

    renderPlot(id, data, layout);
  }, [ id, data, renderPlot, range, scale ]);

  const handleSelect = useCallback((newValue: SelectValue) => {
    setScale(newValue as Scale);
  }, []);

  const chartOptions = (
    <SelectFilter
      enableSearchFilter={false}
      label="Scale"
      showSearch={false}
      value={scale}
      onSelect={handleSelect}>
      {Object.values(Scale).map(scale => (
        <Option key={scale} value={scale}>{capitalize(scale)}</Option>
      ))}
    </SelectFilter>
  );

  return (
    <Section bodyBorder maxHeight options={chartOptions} title={title}>
      <div className={css.base}>
        <div id={id} />
      </div>
    </Section>
  );
};

export default ExperimentChart;
