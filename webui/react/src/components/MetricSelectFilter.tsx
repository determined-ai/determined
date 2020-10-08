import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useMemo, useState } from 'react';

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
  const [ filterInput, setFilterInput ] = useState('');

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

  const totalNumMetrics = useMemo(() => {
    return metricNames.length;
  }, [ metricNames ]);

  const visibleMetrics = useMemo(() => {
    return metricNames.filter((metricName: MetricName) => {
      console.log('Filtering metricName', filterInput, metricName);
      if (metricName.name.includes(filterInput)) {
        return true;
      }
      // if (metricName.type === MetricType.Training && 'training'.includes(filterInput)) {
      //   return true;
      // }
      // if (metricName.type === MetricType.Validation && 'validation'.includes(filterInput)) {
      //   return true;
      // }
      return false;

      // metricName.name !== 'All' && metricName.name !== 'None'
    });
  }, [ metricNames, filterInput ]);

  const handleMetricSelect = useCallback((newValue: SelectValue) => {
    console.log('Metric select event happened', newValue);
    if (!onChange) return;

    let metricName;
    if ((newValue as string) !== 'All') {
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
        // console.log('Metric select event happened - all');
        // console.log('What do I see?', metricNames);
        // const newMetric = metricNames.filter((metricName: MetricName) => metricName.name !== 'All' && metricName.name !== 'None');
        (onChange as MultipleHandler)(visibleMetrics.sort(metricNameSorter));
        setFilterInput('');
      } else {
        const newMetric = Array.isArray(value) ? [ ...value ] : [];
        if (newMetric.indexOf(metricName) === -1) newMetric.push(metricName);
        (onChange as MultipleHandler)(newMetric.sort(metricNameSorter));
        setFilterInput('');
      }

    } else {

      (onChange as SingleHandler)(metricName);
      setFilterInput('');
    }
  }, [ multiple, onChange, value, filterInput ]);

  const handleMetricDeselect = useCallback((newValue: SelectValue) => {
    if (!onChange || !multiple) return;
    if (!Array.isArray(value) || value.length <= 1) return;

    const newMetric = Array.isArray(value) ? [ ...value ] : [];
    const index = newMetric.findIndex(metric => metricNameToValue(metric) === newValue);
    if (index !== -1) newMetric.splice(index, 1);
    (onChange as MultipleHandler)(newMetric.sort(metricNameSorter));
  }, [ multiple, onChange, value ]);

  const handleFiltering = (inputValue: string, option: any) => {
    // if (inputValue !== "") {
    //   setFilterInput(inputValue);
    // }
    if (option.key === 'All') {
      return true;
    } else {
      // TODO: Split on pipe (as long as there is only one) so 'ion|' doesn't return results
      let metricNameOnly = option.key;
      const trainingPrefix = 'training|';
      const validationPrefix = 'validation|';
      let typeForSearch = '';
      if (metricNameOnly.startsWith(trainingPrefix)) {
        metricNameOnly = metricNameOnly.slice(trainingPrefix.length);
        typeForSearch = 'training';
      } else if (metricNameOnly.startsWith(validationPrefix)) {
        metricNameOnly = metricNameOnly.slice(validationPrefix.length);
        typeForSearch = 'validation';
      }

      console.log('handleFiltering', filterInput, option);
      // return metricNameOnly.includes(filterInput) || typeForSearch.includes(filterInput);
      return metricNameOnly.includes(filterInput);
    }
  };

  const handleSearchInputChange = (searchInput: string) => {
    console.log('Search Input', searchInput);
    setFilterInput(searchInput);
  };

  // const handleDropdownVisibleChange = useCallback((open: boolean) => {
  //   if (!open) {
  //     setFilterInput('');
  //   }
  // }, []);

  // const handleClear = useCallback(() => {
  //   setFilterInput('');
  //   let newMetric;
  //   if (validationMetricNames.length > 0) {
  //     newMetric = validationMetricNames[0];
  //   } else if (trainingMetricNames.length > 0){
  //     newMetric = trainingMetricNames[0];
  //   }
  //   if (multiple) {
  //     if (newMetric) {
  //       (onChange as MultipleHandler)([ newMetric ]);
  //     } else {
  //       (onChange as MultipleHandler)([]);
  //     }
  //   } else {
  //     if (newMetric){
  //       (onChange as SingleHandler)(newMetric);
  //     }
  //   }
  // }, [ onChange, validationMetricNames, trainingMetricNames ]);

  const handleClear = useCallback(() => {
    console.log('handleClear');
    setFilterInput('');

    if (multiple) {
      (onChange as MultipleHandler)([ ]);
    }
  }, [ multiple, onChange ]);

  const handleBlur = () => {
    // On blur the inputFilter gets cleared
    console.log('handleBlur');
    setFilterInput('');
  };

  const allSelector = useMemo(() => {
    let allOptionLabel;
    const numVisibleOptions = visibleMetrics.length;
    if (numVisibleOptions === totalNumMetrics) {
      allOptionLabel = 'All';
    } else {
      allOptionLabel =`All ${numVisibleOptions} results`;
    }
    return (
      <Option key={'All'} value={'All'}>
        <BadgeTag label={allOptionLabel} />
      </Option>
    );

  }, [ filterInput, metricNames, visibleMetrics ]);

  const [ maxTagCount, selectorPlaceholder ] = useMemo(() => {
    console.log('metricValues', metricValues);
    if (metricValues === undefined) {
      return [ 0, '' ];
    }
    if (metricValues.length === 0) {
      // return [ -1, `None of ${totalNumMetrics} selected` ];
      return [ -1, 'None selected' ];
    } else if (metricValues.length === totalNumMetrics) {
      return [ 0, `All ${totalNumMetrics} selected` ];
    } else {
      return [ 0, `${metricValues.length} of ${totalNumMetrics} selected` ];
    }
  }, [ metricValues, totalNumMetrics ]);

  return <SelectFilter
    allowClear
    disableTags
    dropdownMatchSelectWidth={400}
    filterOption={handleFiltering}
    label="Metrics"
    maxTagCount={maxTagCount}
    maxTagPlaceholder={selectorPlaceholder}
    mode={multiple ? 'multiple' : undefined}
    showArrow
    style={{ width: 200 }}
    value={metricValues}
    onBlur={handleBlur}
    onClear={handleClear}
    onDeselect={handleMetricDeselect}
    // onDropdownVisibleChange={handleDropdownVisibleChange}
    onSearch={handleSearchInputChange}
    onSelect={handleMetricSelect}>

    {visibleMetrics.length > 1 && allSelector}

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
