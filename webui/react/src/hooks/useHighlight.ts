import { useCallback, useState } from 'react';

import { defaultRowClassName } from 'components/Table';

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

type GetId<RecordType> = (record: RecordType) => number

function useHighlights<RecordType>(getId: GetId<RecordType>): Highlights<RecordType> {

  const [ highlightedId, setHighlightedId ] = useState<number>();

  const handleFocus = useCallback((id: number | null) => {
    setHighlightedId(id ?? undefined);
  }, []);

  const onMouseEnter = useCallback((event: React.MouseEvent, record: RecordType) => {
    if (getId(record)) setHighlightedId(getId(record));
  }, [ getId ]);

  const onMouseLeave: EventDispatcher<RecordType> = useCallback(() => {
    setHighlightedId(undefined);
  }, []);

  const onRow = useCallback((record: RecordType) => ({
    onMouseEnter: (event: React.MouseEvent) => {
      onMouseEnter(event, record);
    },
    onMouseLeave: (event: React.MouseEvent) => {
      onMouseLeave(event, record);
    },
  }), [ onMouseEnter, onMouseLeave ]);

  const rowClassName = useCallback((record: RecordType) => {
    return defaultRowClassName({
      clickable: false,
      highlighted: getId(record) === highlightedId,
    });
  }, [ highlightedId, getId ]);

  return {
    focus: handleFocus,
    id: highlightedId,
    onRow,
    rowClassName,
  };
}

export default useHighlights;
