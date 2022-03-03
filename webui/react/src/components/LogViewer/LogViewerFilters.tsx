import { Button, Select, Space } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useMemo } from 'react';

import MultiSelect from 'components/MultiSelect';
import { LogLevelFromApi } from 'types';
import { alphaNumericSorter } from 'utils/sort';

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
  // sources?: string[],
  // stdtypes?: string[],
}

export const ARIA_LABEL_RESET = 'Reset';

export const LABELS: Record<keyof Filters, string> = {
  agentIds: 'Agent',
  allocationIds: 'Allocation',
  containerIds: 'Container',
  levels: 'Level',
  rankIds: 'Rank',
};

const LogViewerFilters: React.FC<Props> = ({ onChange, onReset, options, values }: Props) => {
  const selectOptions = useMemo(() => {
    const { agentIds, allocationIds, containerIds, rankIds } = options;
    return {
      ...options,
      agentIds: agentIds ? agentIds.sortAll(alphaNumericSorter) : undefined,
      allocationIds: allocationIds ? allocationIds.sortAll(alphaNumericSorter) : undefined,
      containerIds: containerIds ? containerIds.sortAll(alphaNumericSorter) : undefined,
      levels: Object.entries(LogLevelFromApi)
        .filter(entry => entry[1] !== LogLevelFromApi.Unspecified)
        .map(([ key, value ]) => ({ label: key, value })),
      rankIds: rankIds ? rankIds.sortAll(alphaNumericSorter) : undefined,
    };
  }, [ options ]);

  const show = useMemo(() => {
    return Object.keys(selectOptions).reduce((acc, key) => {
      const filterKey = key as keyof Filters;
      const options = selectOptions[filterKey];

      // !! casts `undefined` into the boolean value of `false`.
      acc[filterKey] = !!(options && options.length > 1);

      return acc;
    }, {} as Record<keyof Filters, boolean>);
  }, [ selectOptions ]);

  const isResetShown = useMemo(() => {
    const keys = Object.keys(selectOptions);
    for (let i = 0; i < keys.length; i++) {
      const key = keys[i] as keyof Filters;
      const value = values[key];
      if (value && value.length !== 0) return true;
    }
    return false;
  }, [ selectOptions, values ]);

  const handleChange = useCallback((
    key: keyof Filters,
    caster: NumberConstructor | StringConstructor,
  ) => (value: SelectValue) => {
    onChange?.({ ...values, [key]: (value as Array<string>).map(item => caster(item)) });
  }, [ onChange, values ]);

  const handleReset = useCallback(() => onReset?.(), [ onReset ]);

  return (
    <>
      <Space>
        {show.allocationIds && (
          <MultiSelect
            itemName={LABELS.allocationIds}
            value={values.allocationIds}
            onChange={handleChange('allocationIds', String)}>
            {selectOptions?.allocationIds?.map(id => <Option key={id} value={id}>{id}</Option>)}
          </MultiSelect>
        )}
        {show.agentIds && (
          <MultiSelect
            itemName={LABELS.agentIds}
            value={values.agentIds}
            onChange={handleChange('agentIds', String)}>
            {selectOptions?.agentIds?.map(id => <Option key={id} value={id}>{id}</Option>)}
          </MultiSelect>
        )}
        {show.containerIds && (
          <MultiSelect
            itemName={LABELS.containerIds}
            style={{ width: 150 }}
            value={values.containerIds}
            onChange={handleChange('containerIds', String)}>
            {selectOptions?.containerIds?.map(id => (
              <Option key={id} value={id}>{id || 'No Container'}</Option>
            ))}
          </MultiSelect>
        )}
        {show.rankIds && (
          <MultiSelect
            itemName={LABELS.rankIds}
            value={values.rankIds}
            onChange={handleChange('rankIds', Number)}>
            {selectOptions?.rankIds?.map(id => <Option key={id} value={id}>{id}</Option>)}
          </MultiSelect>
        )}
        <MultiSelect
          itemName={LABELS.levels}
          value={values.levels}
          onChange={handleChange('levels', String)}>
          {selectOptions?.levels.map((level) => (
            <Option key={level.value} value={level.value}>{level.label}</Option>
          ))}
        </MultiSelect>
        {isResetShown && <Button onClick={handleReset}>{ARIA_LABEL_RESET}</Button>}
      </Space>
    </>
  );
};

export default LogViewerFilters;
