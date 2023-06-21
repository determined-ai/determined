import { FilterDropdownProps } from 'antd/lib/table/interface';
import React from 'react';

import TableFilterRange from 'components/Table/TableFilterRange';
import { isNullOrUndefined } from 'utils/data';

import { NumberRangeDict, SetFilters, TrialFilters } from '../Collections/filters';

type FilterPrefix = 'hparams' | 'trainingMetrics' | 'validationMetrics';

export const rangeFilterIsActive = (filter?: NumberRangeDict, key = ''): boolean => {
  const f = filter?.[key];
  return !isNullOrUndefined(f?.min) || !isNullOrUndefined(f?.max);
};

/**
 *
 * @param filterPrefix hparams | validation_metrics | training_metrics
 * @param filters passed down from top
 * @param setFilters passed down from top
 * @returns dropdown filter component
 */
const rangeFilterForPrefix =
  (filterPrefix: FilterPrefix, filters?: TrialFilters, setFilters?: SetFilters) =>
  (key: string): React.FC<FilterDropdownProps> =>
  (filterProps) => {
    const handleRangeApply = (min?: string, max?: string) => {
      setFilters?.((filters: TrialFilters) => {
        const { [filterPrefix]: rangeFilter, ...otherFilters } = filters ?? {};
        if (min || max) {
          const newMin = min || undefined;
          const newMax = max || undefined;
          const newRangeFilter = {
            ...(rangeFilter ?? {}),
            [key]: {
              max: newMax,
              min: newMin,
            },
          };
          return { ...otherFilters, [filterPrefix]: newRangeFilter };
        }
        return otherFilters;
      });
    };

    const handleRangeReset = () => {
      setFilters?.((filters: TrialFilters) => {
        const { [filterPrefix]: rangeFilter, ...otherFilters } = filters ?? {};
        return otherFilters;
      });
    };

    return (
      <TableFilterRange
        {...filterProps}
        max={filters?.[filterPrefix]?.[key]?.max}
        min={filters?.[filterPrefix]?.[key]?.max}
        onReset={handleRangeReset}
        onSet={handleRangeApply}
      />
    );
  };

export default rangeFilterForPrefix;
