import { Select } from 'antd';
import { RefSelectProps, SelectValue } from 'antd/es/select';
import React, { useCallback, useMemo, useRef, useState } from 'react';

import { Metric, MetricType } from 'types';
import { metricKeyToMetric, metricSorter, metricToKey } from 'utils/metric';

import BadgeTag from './BadgeTag';
import MetricBadgeTag from './MetricBadgeTag';
import SelectFilter from 'components/kit/SelectFilter';

const { OptGroup, Option } = Select;
const allOptionId = 'ALL_RESULTS';
const resetOptionId = 'RESET_RESULTS';

type SingleHandler = (value: Metric) => void;
type MultipleHandler = (value: Metric[]) => void;

interface Props {
  defaultMetrics: Metric[];
  dropdownMatchSelectWidth?: number | boolean;
  label?: string;
  metrics: Metric[];
  multiple?: boolean;
  onChange?: SingleHandler | MultipleHandler;
  value?: Metric | Metric[];
  verticalLayout?: boolean;
  width?: number | string;
}

const filterFn = (search: string, metricName: string) => {
  return metricName.toLocaleLowerCase().indexOf(search.toLocaleLowerCase()) !== -1;
};

const MetricSelectFilter: React.FC<Props> = ({
  defaultMetrics,
  dropdownMatchSelectWidth = 400,
  label = 'Metrics',
  metrics,
  multiple,
  value,
  verticalLayout = false,
  width = 200,
  onChange,
}: Props) => {
  const [filterString, setFilterString] = useState('');
  const selectRef = useRef<RefSelectProps>(null);

  const metricValues = useMemo(() => {
    if (multiple && Array.isArray(value)) return value.map((metric) => metricToKey(metric));
    if (!multiple && !Array.isArray(value) && value) return metricToKey(value);
    return undefined;
  }, [multiple, value]);

  const trainingMetrics = useMemo(() => {
    return metrics.filter((metric) => metric.type === MetricType.Training);
  }, [metrics]);

  const validationMetrics = useMemo(() => {
    return metrics.filter((metric) => metric.type === MetricType.Validation);
  }, [metrics]);

  const totalNumMetrics = useMemo(() => {
    return metrics.length;
  }, [metrics]);

  /*
   * visibleMetrics should always match the list of metrics that antd displays to
   * the user, including any filtering.
   */
  const visibleMetrics = useMemo(() => {
    return metrics.filter((metric: Metric) => {
      return filterFn(filterString, metric.name);
    });
  }, [metrics, filterString]);

  const handleMetricSelect = useCallback(
    (newValue: SelectValue) => {
      if (!onChange) return;

      if ((newValue as string) === allOptionId) {
        (onChange as MultipleHandler)(visibleMetrics.sort(metricSorter));
        selectRef.current?.blur();
        return;
      }
      if ((newValue as string) === resetOptionId) {
        (onChange as MultipleHandler)(defaultMetrics.sort(metricSorter));
        selectRef.current?.blur();
        return;
      }

      const metric = metricKeyToMetric(newValue as string);
      if (!metric) return;

      if (multiple) {
        const newMetric = Array.isArray(value) ? [...value] : [];
        if (newMetric.indexOf(metric) === -1) newMetric.push(metric);
        (onChange as MultipleHandler)(newMetric.sort(metricSorter));
      } else {
        (onChange as SingleHandler)(metric);
      }
    },
    [multiple, onChange, value, visibleMetrics, defaultMetrics],
  );

  const handleMetricDeselect = useCallback(
    (newValue: SelectValue) => {
      if (!onChange || !multiple) return;
      if (!Array.isArray(value) || value.length <= 1) return;

      const newMetric = Array.isArray(value) ? [...value] : [];
      const index = newMetric.findIndex((metric) => metricToKey(metric) === newValue);
      if (index !== -1) newMetric.splice(index, 1);
      (onChange as MultipleHandler)(newMetric.sort(metricSorter));
    },
    [multiple, onChange, value],
  );
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const handleFiltering = useCallback((search: string, option: any) => {
    if (option.key === allOptionId || option.key === resetOptionId) return true;
    if (!option.value) return false;

    const metric = metricKeyToMetric(option.value);
    if (metric === undefined) return false;

    return filterFn(search, metric.name);
  }, []);

  const handleSearchInputChange = (searchInput: string) => {
    setFilterString(searchInput);
  };

  const handleBlur = () => {
    setFilterString('');
  };

  const allOption = useMemo(() => {
    let allOptionLabel;
    const numVisibleOptions = visibleMetrics.length;
    if (numVisibleOptions === totalNumMetrics) {
      allOptionLabel = 'All';
    } else {
      allOptionLabel = `All ${numVisibleOptions} results`;
    }
    return (
      <Option key={allOptionId} value={allOptionId}>
        <BadgeTag label={allOptionLabel} />
      </Option>
    );
  }, [totalNumMetrics, visibleMetrics]);

  const [maxTagCount, selectorPlaceholder] = useMemo(() => {
    // This should never happen, but fall back to inoffensive empty placeholder
    if (metricValues === undefined) {
      return [0, ''];
    }
    if (metricValues.length === 0) {
      // If we set maxTagCount=0 in this case, this placeholder will not be displayed.
      return [-1, 'None selected'];
    } else if (metricValues.length === totalNumMetrics) {
      // If we set maxTagCount=-1 in these cases, it will display tags instead of the placeholder.
      return [0, `All ${totalNumMetrics} selected`];
    } else {
      return [0, `${metricValues.length} of ${totalNumMetrics} selected`];
    }
  }, [metricValues, totalNumMetrics]);

  return (
    <SelectFilter
      autoClearSearchValue={false}
      disableTags
      dropdownMatchSelectWidth={dropdownMatchSelectWidth}
      filterOption={handleFiltering}
      label={label}
      maxTagCount={maxTagCount}
      maxTagPlaceholder={selectorPlaceholder}
      mode={multiple ? 'multiple' : undefined}
      ref={selectRef}
      showArrow
      style={{ width }}
      value={metricValues}
      verticalLayout={verticalLayout}
      onBlur={handleBlur}
      onDeselect={handleMetricDeselect}
      onSearch={handleSearchInputChange}
      onSelect={handleMetricSelect}>
      {multiple && visibleMetrics.length > 0 && (
        <Option key={resetOptionId} value={resetOptionId}>
          <BadgeTag label="Reset to Default" />
        </Option>
      )}
      {multiple && visibleMetrics.length > 1 && allOption}
      {validationMetrics.length > 0 && (
        <OptGroup label="Validation Metrics">
          {validationMetrics.map((key) => {
            const value = metricToKey(key);
            return (
              <Option key={value} value={value}>
                <MetricBadgeTag metric={key} />
              </Option>
            );
          })}
        </OptGroup>
      )}
      {trainingMetrics.length > 0 && (
        <OptGroup label="Training Metrics">
          {trainingMetrics.map((key) => {
            const value = metricToKey(key);
            return (
              <Option key={value} value={value}>
                <MetricBadgeTag metric={key} />
              </Option>
            );
          })}
        </OptGroup>
      )}
    </SelectFilter>
  );
};

export default MetricSelectFilter;
