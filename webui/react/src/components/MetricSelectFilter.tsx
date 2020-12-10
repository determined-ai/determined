import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useMemo, useRef, useState } from 'react';

import { MetricName, MetricType } from 'types';
import { metricNameSorter } from 'utils/data';
import { metricNameFromValue, metricNameToValue, valueToMetricName } from 'utils/trial';

import BadgeTag from './BadgeTag';
import SelectFilter from './SelectFilter';

const { OptGroup, Option } = Select;
const allOptionId = 'ALL_RESULTS';
const resetOptionId = 'RESET_RESULTS';

type SingleHandler = (value: MetricName) => void;
type MultipleHandler = (value: MetricName[]) => void;

interface Props {
  defaultMetricNames: MetricName[];
  dropdownMatchSelectWidth?: number | boolean;
  metricNames: MetricName[];
  multiple?: boolean;
  onChange?: SingleHandler | MultipleHandler;
  value?: MetricName | MetricName[];
  verticalLayout?: boolean;
  width?: number | string;
}

const filterFn = (search: string, metricName: string) => {
  return metricName.toLocaleLowerCase().indexOf(search.toLocaleLowerCase()) !== -1;
};

const MetricSelectFilter: React.FC<Props> = ({
  defaultMetricNames,
  dropdownMatchSelectWidth = 400,
  metricNames,
  multiple,
  value,
  verticalLayout = false,
  width = 200,
  onChange,
}: Props) => {
  const [ filterString, setFilterString ] = useState('');
  const selectRef = useRef<Select<SelectValue>>(null);

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

  const totalNumMetrics = useMemo(() => { return metricNames.length; }, [ metricNames ]);

  /*
   * visibleMetrics should always match the list of metrics that antd displays to
   * the user, including any filtering.
   */
  const visibleMetrics = useMemo(() => {
    return metricNames.filter((metricName: MetricName) => {
      return filterFn(filterString, metricName.name);
    });
  }, [ metricNames, filterString ]);

  const handleMetricSelect = useCallback((newValue: SelectValue) => {
    if (!onChange) return;

    if ((newValue as string) === allOptionId) {
      (onChange as MultipleHandler)(visibleMetrics.sort(metricNameSorter));
      selectRef.current?.blur();
      return;
    }
    if ((newValue as string) === resetOptionId) {
      (onChange as MultipleHandler)(defaultMetricNames.sort(metricNameSorter));
      selectRef.current?.blur();
      return;
    }

    const metricName = valueToMetricName(newValue as string);
    if (!metricName) return;

    if (multiple) {
      const newMetric = Array.isArray(value) ? [ ...value ] : [];
      if (newMetric.indexOf(metricName) === -1) newMetric.push(metricName);
      (onChange as MultipleHandler)(newMetric.sort(metricNameSorter));
    } else {
      (onChange as SingleHandler)(metricName);
    }
  }, [ multiple, onChange, value, visibleMetrics, defaultMetricNames ]);

  const handleMetricDeselect = useCallback((newValue: SelectValue) => {
    if (!onChange || !multiple) return;
    if (!Array.isArray(value) || value.length <= 1) return;

    const newMetric = Array.isArray(value) ? [ ...value ] : [];
    const index = newMetric.findIndex(metric => metricNameToValue(metric) === newValue);
    if (index !== -1) newMetric.splice(index, 1);
    (onChange as MultipleHandler)(newMetric.sort(metricNameSorter));
  }, [ multiple, onChange, value ]);

  const handleFiltering = useCallback((search: string, option) => {
    if (option.key === allOptionId || option.key === resetOptionId) {
      return true;
    }
    if (!option.value) {
      /*
       * Handle optionGroups that don't have a value to make TS happy. They aren't
       * impacted by filtering anyway
       */
      return false;
    }
    const metricName = metricNameFromValue(option.value);
    if (metricName === undefined) {
      /*
       * Handle metric values that don't start with 'training|' or 'validation|'. This
       * shouldn't ever happen and metricNameFromValue logs an error if it does.
       */
      return false;
    }
    return filterFn(search, metricName.name);
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
      allOptionLabel =`All ${numVisibleOptions} results`;
    }
    return (
      <Option key={allOptionId} value={allOptionId}>
        <BadgeTag label={allOptionLabel} />
      </Option>
    );

  }, [ totalNumMetrics, visibleMetrics ]);

  const [ maxTagCount, selectorPlaceholder ] = useMemo(() => {
    // This should never happen, but fall back to inoffensive empty placeholder
    if (metricValues === undefined) {
      return [ 0, '' ];
    }
    if (metricValues.length === 0) {
      // If we set maxTagCount=0 in this case, this placeholder will not be displayed.
      return [ -1, 'None selected' ];
    } else if (metricValues.length === totalNumMetrics) {
      // If we set maxTagCount=-1 in these cases, it will display tags instead of the placeholder.
      return [ 0, `All ${totalNumMetrics} selected` ];
    } else {
      return [ 0, `${metricValues.length} of ${totalNumMetrics} selected` ];
    }
  }, [ metricValues, totalNumMetrics ]);

  return <SelectFilter
    autoClearSearchValue={false}
    disableTags
    dropdownMatchSelectWidth={dropdownMatchSelectWidth}
    filterOption={handleFiltering}
    label="Metrics"
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

    { multiple && visibleMetrics.length > 0 &&
    <Option key={resetOptionId} value={resetOptionId}>
      <BadgeTag label='Reset to Default' />
    </Option>}

    { multiple && visibleMetrics.length > 1 && allOption}

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
