import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useMemo } from 'react';

import { MetricName, MetricType } from 'types';
import { metricNameSorter } from 'utils/data';
import { metricNameToValue, valueToMetricName } from 'utils/trial';

import BadgeTag from './BadgeTag';
import SelectFilter from './SelectFilter';

const { OptGroup, Option } = Select;

type SingleHandler = (value: MetricName) => void;
type MultipleHandler = (value: MetricName[]) => void;

interface Props {
  metricNames: MetricName[];
  multiple?: boolean;
  onChange?: SingleHandler | MultipleHandler;
  value?: MetricName | MetricName[];
}

const MetricSelectFilter: React.FC<Props> = ({ metricNames, multiple, onChange, value }: Props) => {

  const metricValues = useMemo(() => {
    if (multiple && Array.isArray(value)) return value.map(metric => metricNameToValue(metric));
    if (!multiple && !Array.isArray(value) && value) return metricNameToValue(value);
    return undefined;
  }, [ multiple, value ]);

  const trainingMetricNames = useMemo(() => {
    return metricNames.filter(metric => metric.type === MetricType.Training);
  }, [ metricNames ]);

  const validationMetricNames = useMemo(() => {
    return metricNames.filter(metric => metric.type === MetricType.Validation);
  }, [ metricNames ]);

  const handleMetricSelect = useCallback((newValue: SelectValue) => {
    if (!onChange) return;

    let metricName;
    if ((newValue as string) !== 'All' && (newValue as string) !== 'None') {
      metricName = valueToMetricName(newValue as string);
      if (!metricName) return;
    } else {
      metricName = {
        name: newValue as string,
        type: MetricType.Placeholder,
      };
    }

    if (multiple) {
      if (newValue === 'All') {
        const newMetric = metricNames.filter((metricName: MetricName) => metricName.name !== 'All' && metricName.name !== 'None');
        (onChange as MultipleHandler)(newMetric.sort(metricNameSorter));
      } else if (newValue === 'None') {
        const newMetric: MetricName[] = [];
        (onChange as MultipleHandler)(newMetric.sort(metricNameSorter));
      } else {
        const newMetric = Array.isArray(value) ? [ ...value ] : [];
        if (newMetric.indexOf(metricName) === -1) newMetric.push(metricName);
        (onChange as MultipleHandler)(newMetric.sort(metricNameSorter));
      }

    } else {

      (onChange as SingleHandler)(metricName);
    }
  }, [ multiple, onChange, value ]);

  const handleMetricDeselect = useCallback((newValue: SelectValue) => {
    if (!onChange || !multiple) return;
    if (!Array.isArray(value) || value.length <= 1) return;

    const newMetric = Array.isArray(value) ? [ ...value ] : [];
    const index = newMetric.findIndex(metric => metricNameToValue(metric) === newValue);
    if (index !== -1) newMetric.splice(index, 1);
    (onChange as MultipleHandler)(newMetric.sort(metricNameSorter));
  }, [ multiple, onChange, value ]);

  return <SelectFilter
    disableTags
    dropdownMatchSelectWidth={400}
    label="Metrics"
    mode={multiple ? 'multiple' : undefined}
    showArrow
    style={{ width: 150 }}
    value={metricValues}
    onDeselect={handleMetricDeselect}
    onSelect={handleMetricSelect}>
    {multiple && <OptGroup label="Convenience (TODO: Better Name)">
      {[ 'All', 'None' ].map(key => {
        return <Option key={key} value={key}>
          <BadgeTag label={key} />
        </Option>;
      })}
    </OptGroup>}

    {validationMetricNames.length > 0 && <OptGroup label="Validation Metrics">
      {validationMetricNames.map(key => {
        const value = metricNameToValue(key);
        return <Option key={value} value={value}>
          <BadgeTag
            label={key.name}
            tooltip={key.type}>{key.type.substr(0, 1).toUpperCase()}</BadgeTag>
        </Option>;
      })}
    </OptGroup>}
    {trainingMetricNames.length > 0 && <OptGroup label="Training Metrics">
      {trainingMetricNames.map(key => {
        const value = metricNameToValue(key);
        return <Option key={value} value={value}>
          <BadgeTag
            label={key.name}
            tooltip={key.type}>{key.type.substr(0, 1).toUpperCase()}</BadgeTag>
        </Option>;
      })}
    </OptGroup>}

  </SelectFilter>;
};

export default MetricSelectFilter;
