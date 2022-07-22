import { Pagination, Tooltip } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';

import { keyEmitter, KeyEvent } from 'hooks/useKeyTracker';

import css from './RoutePagination.module.scss';

interface Props {
  currentId: number;
  ids: number[];
  onSelectId: (selectedId: number) => void;
  tooltipLabel?: string;
}

const RoutePagination: React.FC<Props> =
({ currentId, ids, onSelectId, tooltipLabel }: Props) => {
  const [ currentPage, setCurrentPage ] = useState<number>(0);
  const navigateToId = useCallback((page: number) => {
    onSelectId(ids[page - 1]);
  }, [ ids, onSelectId ]);

  useEffect(() => {
    const keyUpListener = (e: KeyboardEvent) => {
      if (e.key === 'ArrowLeft' && currentPage > 1) {
        navigateToId(currentPage - 1);
      } else if (e.key === 'ArrowRight' && currentPage < ids.length) {
        navigateToId(currentPage + 1);
      }
    };

    keyEmitter.on(KeyEvent.KeyUp, keyUpListener);

    return () => {
      keyEmitter.off(KeyEvent.KeyUp, keyUpListener);
    };
  }, [ currentPage, ids, navigateToId ]);

  useEffect(() => {
    const idx = ids.findIndex((i: number) => i === currentId);
    setCurrentPage(idx + 1);
  }, [ ids, currentId, setCurrentPage ]);

  return (
    <div className={css.base}>
      <Pagination
        current={currentPage}
        itemRender={(page, type, originalElement) => {
          if (tooltipLabel &&
            (type === 'prev' && currentPage > 1) ||
            (type === 'next' && currentPage < ids.length)) {
            return (
              <Tooltip
                placement="bottom"
                title={`${type === 'prev' ? 'Previous' : 'Next'} ${tooltipLabel}`}>
                {originalElement}
              </Tooltip>
            );
          } else {
            return originalElement;
          }
        }}
        pageSize={1}
        showSizeChanger={false}
        total={ids.length}
        onChange={navigateToId}
      />
    </div>
  );
};

export default RoutePagination;
