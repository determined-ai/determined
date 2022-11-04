import { useCallback, useState } from 'react';

import { defaultRowClassName } from 'components/Table/Table';

type EventDispatcher<RecordType> = (event: React.MouseEvent, record: RecordType) => void;

export interface Highlights<RecordType> {
  focus: (id: number | null) => void;
  id: number | undefined;
  onRow: (record: RecordType) => {
    onMouseEnter: (event: React.MouseEvent) => void;
    onMouseLeave: (event: React.MouseEvent) => void;
  };
  rowClassName: (record: RecordType) => string;
}

type GetId<RecordType> = (record: RecordType) => number;

/**
 * This hook encapsulates the functionality of synced highlighting between table(s)
 * other representations, as we have beteween the HpTrialTable and ExperimentVisualization
 * Currently this hook is only being used for Trials Comparison table, but we can migrate
 * the Multi-trial overview page to use it as well
 */
function useHighlights<RecordType>(getId: GetId<RecordType>): Highlights<RecordType> {
  const [highlightedId, setHighlightedId] = useState<number>();

  const handleFocus = useCallback((id: number | null) => {
    setHighlightedId(id ?? undefined);
  }, []);

  const onMouseEnter = useCallback(
    (event: React.MouseEvent, record: RecordType) => {
      if (getId(record)) setHighlightedId(getId(record));
    },
    [getId],
  );

  const onMouseLeave: EventDispatcher<RecordType> = useCallback(() => {
    setHighlightedId(undefined);
  }, []);

  const onRow = useCallback(
    (record: RecordType) => ({
      onMouseEnter: (event: React.MouseEvent) => {
        onMouseEnter(event, record);
      },
      onMouseLeave: (event: React.MouseEvent) => {
        onMouseLeave(event, record);
      },
    }),
    [onMouseEnter, onMouseLeave],
  );

  const rowClassName = useCallback(
    (record: RecordType) => {
      return defaultRowClassName({
        clickable: false,
        highlighted: getId(record) === highlightedId,
      });
    },
    [highlightedId, getId],
  );

  return {
    focus: handleFocus,
    id: highlightedId,
    onRow,
    rowClassName,
  };
}

export default useHighlights;
