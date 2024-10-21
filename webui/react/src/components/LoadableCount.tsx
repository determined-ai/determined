import Button from 'hew/Button';
import { Loadable } from 'hew/utils/loadable';
import { useMemo } from 'react';

import useMobile from 'hooks/useMobile';
import { pluralizer } from 'utils/string';

import css from './LoadableCount.module.scss';

interface Props {
  total: Loadable<number>;
  labelSingular: string;
  labelPlural: string;
  onActualSelectAll?: () => void;
  onClearSelect?: () => void;
  pageSize?: number;
  selectedCount: number;
}

const LoadableCount: React.FC<Props> = ({
  total,
  labelPlural,
  labelSingular,
  onActualSelectAll,
  onClearSelect,
  pageSize = 20,
  selectedCount,
}: Props) => {
  const isMobile = useMobile();

  const selectionLabel = useMemo(() => {
    return Loadable.match(total, {
      Failed: () => null,
      Loaded: (loadedTotal) => {
        let label = `${loadedTotal.toLocaleString()} ${pluralizer(
          loadedTotal,
          labelSingular.toLowerCase(),
          labelPlural.toLowerCase(),
        )}`;

        if (selectedCount) {
          label = `${selectedCount.toLocaleString()} of ${label} selected`;
        }

        return label;
      },
      NotLoaded: () => `Loading ${labelPlural.toLowerCase()}...`,
    });
  }, [labelPlural, labelSingular, total, selectedCount]);

  const actualSelectAll = useMemo(() => {
    return Loadable.match(total, {
      _: () => null,
      Loaded: (loadedTotal) => {
        if (onActualSelectAll && selectedCount >= pageSize && selectedCount < loadedTotal) {
          return (
            <Button type="text" onClick={onActualSelectAll}>
              Select all {labelPlural} in table
            </Button>
          );
        } else if (onClearSelect && selectedCount >= pageSize) {
          return (
            <Button type="text" onClick={onClearSelect}>
              Clear Selection
            </Button>
          );
        }

        return null;
      },
    });
  }, [labelPlural, onActualSelectAll, onClearSelect, pageSize, selectedCount, total]);

  if (!isMobile) {
    return (
      <>
        <span className={css.base} data-test="count">
          {selectionLabel}
        </span>
        {actualSelectAll}
      </>
    );
  } else {
    return null;
  }
};

export default LoadableCount;
