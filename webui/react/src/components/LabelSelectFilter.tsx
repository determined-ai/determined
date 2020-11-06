import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import { getAllExperimentLabels } from 'services/api';
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

  const labelValues = useMemo(() => {
    if (Array.isArray(value)) {
      return value.map(label => label.toString());
    }

    return undefined;
  }, [ value ]);

  const handleLabelSelect = useCallback((newValue: SelectValue) => {
    if (!onChange) return;

    const newLabel = newValue.toString();
    if (newLabel === ALL_VALUE) {
      onChange([]);
      if (document && document.activeElement) {
        (document.activeElement as HTMLElement).blur();
      }
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
      const labels = await getAllExperimentLabels();
      setLabels([
        ALL_VALUE,
        ...labels
          .sort((a, b) => alphanumericSorter(a, b)),
      ]);
    };

    fetchData();
  }, []);

  const options = useMemo(() => {
    const list: React.ReactNode[] = [ ];

    labels.map(label => {
      list.push(<Option key={label} value={label}>{label}</Option>);
    });

    return list;
  }, [ labels ]);

  return <SelectFilter
    disableTags
    dropdownMatchSelectWidth={200}
    label="Labels"
    mode="multiple"
    placeholder={'All'}
    showArrow
    style={{ width: 130 }}
    value={labelValues}
    onDeselect={handleLabelDeselect}
    onSelect={handleLabelSelect}
  >
    {options}
  </SelectFilter>;
};

export default LabelSelectFilter;
