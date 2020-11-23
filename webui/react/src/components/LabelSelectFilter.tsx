import React, { useEffect, useState } from 'react';

import { getExperimentLabels } from 'services/api';
import { alphanumericSorter } from 'utils/data';

import MultiSelect from './MultiSelect';

interface Props {
  onChange?: (value: (number|string)[]) => void;
  value?: string[];
}

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
    <MultiSelect
      label="Labels"
      options={labels}
      value={value || []}
      onChange={onChange}
    />
  );
};

export default LabelSelectFilter;
