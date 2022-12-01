import { Button, Input, Select, Space } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useMemo } from 'react';

import { alphaNumericSorter } from 'shared/utils/sort';
import { LogLevelFromApi } from 'types';

const { Option } = Select;

interface Props {
  onChange?: (filters: Filters) => void;
  onReset?: () => void;
  options: Filters;
  showSearch: boolean;
  values: Filters;
}

export interface Filters {
  agentIds?: string[];
  allocationIds?: string[];
  containerIds?: string[];
  levels?: LogLevelFromApi[];
  rankIds?: number[];
  searchText?: string;
  // sources?: string[],
  // stdtypes?: string[],
}

export const ARIA_LABEL_RESET = 'Reset';

export const LABELS: Record<keyof Filters, string> = {
  agentIds: 'Agents',
  allocationIds: 'Allocations',
  containerIds: 'Containers',
  levels: 'Levels',
  rankIds: 'Ranks',
  searchText: 'Searches',
};

const LogViewerFilters: React.FC<Props> = ({
  onChange,
  onReset,
  options,
  showSearch,
  values,
}: Props) => {
  const selectOptions = useMemo(() => {
    const { agentIds, allocationIds, containerIds, rankIds } = options;
    return {
      ...options,
      agentIds: agentIds ? agentIds.sortAll(alphaNumericSorter) : undefined,
      allocationIds: allocationIds ? allocationIds.sortAll(alphaNumericSorter) : undefined,
      containerIds: containerIds ? containerIds.sortAll(alphaNumericSorter) : undefined,
      levels: Object.entries(LogLevelFromApi)
        .filter((entry) => entry[1] !== LogLevelFromApi.Unspecified)
        .map(([key, value]) => ({ label: key, value })),
      rankIds: rankIds ? [-100].concat(rankIds.sortAll(alphaNumericSorter)) : [-100],
    };
  }, [options]);

  const moreThanOne = useMemo(() => {
    return Object.keys(selectOptions).reduce((acc, key) => {
      const filterKey = key as keyof Filters;
      const options = selectOptions[filterKey];

      // !! casts `undefined` into the boolean value of `false`.
      acc[filterKey] = !!(options && options.length > 1) || filterKey === 'rankIds';

      return acc;
    }, {} as Record<keyof Filters, boolean>);
  }, [selectOptions]);

  const isResetShown = useMemo(() => {
    const keys = Object.keys(selectOptions);
    for (let i = 0; i < keys.length; i++) {
      const key = keys[i] as keyof Filters;
      const value = values[key];
      if (value && value.length !== 0) return true;
    }
    return false;
  }, [selectOptions, values]);

  const handleChange = useCallback(
    (key: keyof Filters, caster: NumberConstructor | StringConstructor) => (value: SelectValue) => {
      onChange?.({ ...values, [key]: (value as Array<string>).map((item) => caster(item)) });
    },
    [onChange, values],
  );

  const handleSearch = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) =>
      onChange?.({ ...values, searchText: e.target.value }),
    [onChange, values],
  );

  const handleReset = useCallback(() => onReset?.(), [onReset]);

  return (
    <>
      <Space>
        {showSearch && (
          <Input placeholder="Search Logs..." value={values.searchText} onChange={handleSearch} />
        )}
        {moreThanOne.allocationIds && (
          <Select
            maxTagCount="responsive"
            mode="multiple"
            placeholder={`All ${LABELS.allocationIds}`}
            style={{ width: 150 }}
            value={values.allocationIds}
            onChange={handleChange('allocationIds', String)}>
            {selectOptions?.allocationIds?.map((id, index) => (
              <Option key={id || `no-id-${index}`} value={id}>
                {id || 'No Allocation ID'}
              </Option>
            ))}
          </Select>
        )}
        {moreThanOne.agentIds && (
          <Select
            maxTagCount="responsive"
            mode="multiple"
            placeholder={`All ${LABELS.agentIds}`}
            style={{ width: 150 }}
            value={values.agentIds}
            onChange={handleChange('agentIds', String)}>
            {selectOptions?.agentIds?.map((id, index) => (
              <Option key={id || `no-id-${index}`} value={id}>
                {id || 'No Agent ID'}
              </Option>
            ))}
          </Select>
        )}
        {moreThanOne.containerIds && (
          <Select
            maxTagCount="responsive"
            mode="multiple"
            placeholder={`All ${LABELS.containerIds}`}
            style={{ width: 150 }}
            value={values.containerIds}
            onChange={handleChange('containerIds', String)}>
            {selectOptions?.containerIds?.map((id, index) => (
              <Option key={id || `no-id-${index}`} value={id}>
                {id || 'No Container ID'}
              </Option>
            ))}
          </Select>
        )}
        {moreThanOne.rankIds && (
          <Select
            maxTagCount="responsive"
            mode="multiple"
            placeholder={`All ${LABELS.rankIds}`}
            style={{ width: 150 }}
            value={values.rankIds}
            onChange={handleChange('rankIds', Number)}>
            {selectOptions?.rankIds?.map((id, index) => (
              <Option key={id ?? `no-id-${index}`} value={id}>
                {id === -100 ? 'No Rank' : id}
              </Option>
            ))}
          </Select>
        )}
        <Select
          maxTagCount="responsive"
          mode="multiple"
          placeholder={`All ${LABELS.levels}`}
          style={{ width: 150 }}
          value={values.levels}
          onChange={handleChange('levels', String)}>
          {selectOptions?.levels.map((level) => (
            <Option key={level.value} value={level.value}>
              {level.label}
            </Option>
          ))}
        </Select>
        {isResetShown && <Button onClick={handleReset}>{ARIA_LABEL_RESET}</Button>}
      </Space>
    </>
  );
};

export default LogViewerFilters;
