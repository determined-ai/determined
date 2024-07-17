import { Loadable } from 'hew/utils/loadable';
import { useMemo } from 'react';

import useMobile from 'hooks/useMobile';
import { pluralizer } from 'utils/string';

import css from './LoadableCount.module.scss';

interface Props {
  total: Loadable<number>;
  labelSingular: string;
  labelPlural: string;
  selectedCount: number;
}

const LoadableCount: React.FC<Props> = ({
  total,
  labelPlural,
  labelSingular,
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
          labelPlural,
        )}`;

        if (selectedCount) {
          label = `${selectedCount} of ${label} selected`;
        }

        return label;
      },
      NotLoaded: () => `Loading ${labelPlural.toLowerCase()}...`,
    });
  }, [labelPlural, labelSingular, total, selectedCount]);

  if (!isMobile) {
    return (
      <span className={css.expNum} data-test="expNum">
        {selectionLabel}
      </span>
    );
  } else {
    return null;
  }
};

export default LoadableCount;
