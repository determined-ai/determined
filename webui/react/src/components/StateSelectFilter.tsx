import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useMemo } from 'react';

import { ALL_VALUE, CommandState, RunState } from 'types';
import { commandStateToLabel, runStateToLabel } from 'utils/types';

import SelectFilter from './SelectFilter';

const { OptGroup, Option } = Select;

interface Props {
  onChange?: (value: SelectValue) => void;
  showCommandStates?: boolean;
  showExperimentStates?: boolean;
  value?: SelectValue;
}

const defaultProps = {
  showCommandStates: true,
  showExperimentStates: true,
};

const commandOptions = Object.values(CommandState).map((value) => (
  <Option key={value} value={value}>{commandStateToLabel[value]}</Option>
));

const experimentOptions = Object.values(RunState).map((value) => (
  <Option key={value} value={value}>{runStateToLabel[value]}</Option>
));

const StateSelectFilter: React.FC<Props> = ({
  onChange,
  showCommandStates,
  showExperimentStates,
  value,
}: Props) => {
  const options = useMemo(() => {
    if (showExperimentStates && showCommandStates) {
      return <>
        <OptGroup key="experimentGroup" label="Experiment States">{experimentOptions}</OptGroup>
        <OptGroup key="commandGroup" label="Command States">{commandOptions}</OptGroup>
      </>;
    } else if (showExperimentStates) {
      return experimentOptions;
    } else if (showCommandStates) {
      return commandOptions;
    }
  }, [ showExperimentStates, showCommandStates ]);

  const handleSelect = useCallback((newValue: SelectValue) => {
    if (!onChange) return;
    const singleValue = Array.isArray(newValue) ? newValue[0] : newValue;
    onChange(singleValue);
  }, [ onChange ]);

  return (
    <SelectFilter
      dropdownMatchSelectWidth={150}
      label="State"
      value={value}
      onSelect={handleSelect}
    >
      <Option key={ALL_VALUE} value={ALL_VALUE}>All</Option>
      {options}
    </SelectFilter>
  );
};

StateSelectFilter.defaultProps = defaultProps;

export default StateSelectFilter;
