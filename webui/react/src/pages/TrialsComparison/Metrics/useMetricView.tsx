import React, { ReactNode, useCallback, useEffect, useReducer, useState } from 'react';

import MetricSelectFilter from 'components/MetricSelectFilter';
import ScaleSelectFilter from 'components/ScaleSelectFilter';
import { Metric, MetricType, Scale } from 'types';

export enum Layout {
  Grid = 'grid',
  List = 'list',
}

export interface MetricView {
  layout: Layout;
  metric: Metric;
  scale: Scale;
}

interface MetricViewSelectProps {
  metrics: Metric[];
  onChange?: (view: MetricView) => void;
  onReset?: () => void;
  view: MetricView;
}

enum ActionType {
  Set,
  SetMetric,
  SetLayout,
  SetScale,
}
type Action =
| { type: ActionType.Set; value: MetricView }
| { type: ActionType.SetMetric; value: Metric }
| { type: ActionType.SetLayout; value: Layout }
| { type: ActionType.SetScale; value: Scale }

const reducer = (state: MetricView, action: Action) => {
  switch (action.type) {
    case ActionType.SetMetric:
      return { ...state, metric: action.value };
    case ActionType.SetLayout:
      return { ...state, view: action.value };
    case ActionType.SetScale:
      return { ...state, scale: action.value };
    default:
      return state;
  }
};

export const MetricViewSelect: React.FC<MetricViewSelectProps> = ({
  view,
  metrics,
  onChange,
}) => {
  const [ localView, dispatch ] = useReducer(reducer, view);

  const handleScaleChange = useCallback((scale: Scale) => {
    dispatch({ type: ActionType.SetScale, value: scale });
  }, []);

  const handleMetricChange = useCallback((metric: Metric) => {
    dispatch({ type: ActionType.SetMetric, value: metric });
  }, []);

  useEffect(() => {
    if (onChange) onChange(localView);
  }, [ localView, onChange ]);

  return (
    <>
      <MetricSelectFilter
        defaultMetrics={metrics}
        label="Metric"
        metrics={metrics}
        multiple={false}
        value={localView.metric}
        width={'100%'}
        onChange={handleMetricChange}
      />
      <ScaleSelectFilter value={localView.scale} onChange={handleScaleChange} />
    </>
  );
};

interface MetricViewInterface {
  controls: ReactNode;
  view?: MetricView;

}

const useMetricView = (metrics: Metric[]): MetricViewInterface => {

  const [ view, setView ] = useState<MetricView>();

  const handleViewChange = useCallback((view: MetricView) => {
    setView(view);
  }, []);

  useEffect(() => {
    if (!view && metrics.length) {
      const defaultMetric = metrics
        .filter((m) => m.type === MetricType.Validation)[0]
        ?? metrics[0];
      setView({ layout: Layout.Grid, metric: defaultMetric, scale: Scale.Linear });
    }
  }, [ view, metrics ]);

  const controls = (
    view && (
      <MetricViewSelect
        metrics={metrics}
        view={view}
        onChange={handleViewChange}
      />
    )
  );

  return { controls, view };
};

export default useMetricView;
