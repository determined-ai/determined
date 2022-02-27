import { Button, Select, Space } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useMemo } from 'react';

import MultiSelect from 'components/MultiSelect';
import { LogLevelFromApi } from 'types';

const { Option } = Select;

interface Props {
  onChange?: (filters: Filters) => void;
  onReset?: () => void;
  options: Filters;
  values: Filters;
}

export interface Filters {
  agentIds?: string[],
  allocationIds?: string[],
  containerIds?: string[],
  levels?: LogLevelFromApi[],
  rankIds?: number[],
  sources?: string[],
  stdtypes?: string[],
}

const TaskLogFilters: React.FC<Props> = ({ onChange, onReset, options, values }: Props) => {
  const selectOptions = useMemo(() => {
    return {
      ...options,
      levels: Object.entries(LogLevelFromApi)
        .filter(entry => entry[1] !== LogLevelFromApi.Unspecified)
        .map(([ key, value ]) => ({ label: key, value })),
    };
  }, [ options ]);

  const handleChange = useCallback((
    key: keyof Filters,
    caster: NumberConstructor | StringConstructor,
  ) => (value: SelectValue) => {
    onChange?.({
      ...values,
      [key]: (value as Array<string>).map(item => caster(item)),
    });
  }, [ onChange, values ]);

  const handleReset = useCallback(() => onReset?.(), [ onReset ]);

  return (
    <>
      <Space>
        {selectOptions?.allocationIds?.length !== 0 && (
          <MultiSelect
            itemName="Allocation"
            value={values.allocationIds}
            onChange={handleChange('allocationIds', String)}>
            {selectOptions?.allocationIds?.map(id => <Option key={id} value={id}>{id}</Option>)}
          </MultiSelect>
        )}
        {selectOptions?.agentIds?.length !== 0 && (
          <MultiSelect
            itemName="Agent"
            value={values.agentIds}
            onChange={handleChange('agentIds', String)}>
            {selectOptions?.agentIds?.map(id => <Option key={id} value={id}>{id}</Option>)}
          </MultiSelect>
        )}
        {selectOptions?.containerIds?.length !== 0 && (
          <MultiSelect
            itemName="Container"
            style={{ width: 150 }}
            value={values.containerIds}
            onChange={handleChange('containerIds', String)}>
            {selectOptions?.containerIds?.map(id => (
              <Option key={id} value={id}>{id || 'No Container'}</Option>
            ))}
          </MultiSelect>
        )}
        {selectOptions?.rankIds?.length !== 0 && (
          <MultiSelect
            itemName="Rank"
            value={values.rankIds}
            onChange={handleChange('rankIds', Number)}>
            {selectOptions?.rankIds?.map(id => <Option key={id} value={id}>{id}</Option>)}
          </MultiSelect>
        )}
        <MultiSelect
          itemName="Level"
          value={values.levels}
          onChange={handleChange('levels', String)}>
          {selectOptions?.levels.map((level) => (
            <Option key={level.value} value={level.value}>{level.label}</Option>
          ))}
        </MultiSelect>
        <Button onClick={handleReset}>Reset</Button>
      </Space>
    </>
  );
};

export default TaskLogFilters;
