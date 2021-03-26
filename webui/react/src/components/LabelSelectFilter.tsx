import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useEffect, useState } from 'react';

import { getExperimentLabels } from 'services/api';
import { alphanumericSorter } from 'utils/sort';

import MultiSelect from './MultiSelect';

interface Props {
  onChange?: (newValue: SelectValue) => void;
  value?: string[];
}

const { Option } = Select;

const LabelSelectFilter: React.FC<Props> = ({ onChange, value }: Props) => {
  const [ labels, setLabels ] = useState<string[]>([]);
  const [ canceler ] = useState(new AbortController());

  useEffect(() => {
    const fetchData = async () => {
      try {
        const labels = await getExperimentLabels({ signal: canceler.signal });
        setLabels(
          labels.sort((a, b) => alphanumericSorter(a, b)),
        );
      } catch (e) {}
    };

    fetchData();

    return () => canceler.abort();
  }, [ canceler ]);

  return (
    <MultiSelect label="Labels" value={value} onChange={onChange}>
      {labels.map((label) => <Option key={label} value={label}>{label}</Option>)}
    </MultiSelect>
  );
};

export default LabelSelectFilter;
