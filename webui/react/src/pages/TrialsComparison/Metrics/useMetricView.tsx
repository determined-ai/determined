import React, { ReactNode, useCallback, useEffect, useReducer, useState } from 'react';

import MetricSelect from 'components/MetricSelect';
import ScaleSelect from 'components/ScaleSelect';
import { ValueOf } from 'types';
import { Metric, MetricType, Scale } from 'types';

export const Layout = {
  Grid: 'grid',
  List: 'list',
} as const;

export type Layout = ValueOf<typeof Layout>;

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

const ActionType = {
  Set: 0,
  SetLayout: 1,
  SetMetric: 2,
  SetScale: 3,
} as const;

type Action =
  | { type: typeof ActionType.Set; value: MetricView }
  | { type: typeof ActionType.SetMetric; value: Metric }
  | { type: typeof ActionType.SetLayout; value: Layout }
  | { type: typeof ActionType.SetScale; value: Scale };

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

export const MetricViewSelect: React.FC<MetricViewSelectProps> = ({ view, metrics, onChange }) => {
  const [localView, dispatch] = useReducer(reducer, view);

  const handleScaleChange = useCallback((scale: Scale) => {
    dispatch({ type: ActionType.SetScale, value: scale });
  }, []);

  const handleMetricChange = useCallback((metric: Metric) => {
    dispatch({ type: ActionType.SetMetric, value: metric });
  }, []);

  useEffect(() => {
    if (onChange) onChange(localView);
  }, [localView, onChange]);

  return (
    <>
      <MetricSelect
        defaultMetrics={metrics}
        label="Metric"
        metrics={metrics}
        multiple={false}
        value={localView.metric}
        onChange={handleMetricChange}
      />
      <ScaleSelect value={localView.scale} onChange={handleScaleChange} />
    </>
  );
};

interface MetricViewInterface {
  controls: ReactNode;
  view?: MetricView;
}

const useMetricView = (metrics: Metric[]): MetricViewInterface => {
  const [view, setView] = useState<MetricView>();

  const handleViewChange = useCallback((view: MetricView) => {
    setView(view);
  }, []);

  useEffect(() => {
    if (!view && metrics.length) {
      const defaultMetric =
        metrics.filter((m) => m.type === MetricType.Validation)[0] ?? metrics[0];
      setView({ layout: Layout.Grid, metric: defaultMetric, scale: Scale.Linear });
    }
  }, [view, metrics]);

  const controls = view && (
    <MetricViewSelect metrics={metrics} view={view} onChange={handleViewChange} />
  );

  return { controls, view };
};

export default useMetricView;
