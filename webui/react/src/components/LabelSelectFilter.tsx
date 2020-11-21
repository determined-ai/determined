import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import { getExperimentLabels } from 'services/api';
import { alphanumericSorter } from 'utils/data';

import SelectFilter from './SelectFilter';

interface Props {
  onChange?: (value: string[]) => void;
  value?: string[];
}

const ALL_VALUE = 'All';

const { Option } = Select;

const LabelSelectFilter: React.FC<Props> = ({ onChange, value }: Props) => {
  const [ labels, setLabels ] = useState<string[]>([]);
  const [ canceler ] = useState(new AbortController());

  const labelValues = useMemo(() => {
    if (!Array.isArray(value)) return;
    return value.map(label => label.toString());
  }, [ value ]);

  const options = useMemo(() => {
    const list: React.ReactNode[] = [ ];

    labels.map(label => {
      list.push(<Option key={label} value={label}>{label}</Option>);
    });

    return list;
  }, [ labels ]);

  const handleLabelSelect = useCallback((newValue: SelectValue) => {
    if (!onChange) return;

    const newLabel = newValue.toString();
    if (newLabel === ALL_VALUE) {
      onChange([]);
      const activeElement = document && document.activeElement as HTMLElement;
      if (activeElement) activeElement.blur();
      return;
    }

    const labelList = Array.isArray(value) ? [ ...value ] : [];
    if (labelList.indexOf(newLabel) === -1) labelList.push(newLabel);
    onChange(labelList);
  }, [ onChange, value ]);

  const handleLabelDeselect = useCallback((newValue: SelectValue) => {
    if (!onChange) return;
    const labelList = Array.isArray(value) ? [ ...value ] : [];
    const newLabel = newValue.toString();
    const index = labelList.indexOf(newLabel);
    if (index !== -1) labelList.splice(index, 1);
    onChange(labelList);
  }, [ onChange, value ]);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const labels = await getExperimentLabels({ signal: canceler.signal });
        setLabels([
          ALL_VALUE,
          ...labels.sort((a, b) => alphanumericSorter(a, b)),
        ]);
      } catch (e) {}
    };

    fetchData();

    return () => canceler.abort();
  }, [ canceler ]);

  return (
    <SelectFilter
      disableTags
      dropdownMatchSelectWidth={200}
      label="Labels"
      mode="multiple"
      placeholder={'All'}
      showArrow
      style={{ width: 130 }}
      value={labelValues}
      onDeselect={handleLabelDeselect}
      onSelect={handleLabelSelect}>
      {options}
    </SelectFilter>
  );
};

export default LabelSelectFilter;
