import { Pagination } from 'antd';
import React, { ReactNode } from 'react';

interface PaginationProps {
  current?: number;
  itemRender?: (page: number, type: 'page' | 'prev' | 'next' | 'jump-prev' | 'jump-next', originalElement: ReactNode) => ReactNode;
  onChange?: () => void;
  pageSize?: number;
  showSizeChanger?: boolean;
  total: number;
}

const PaginationComponent: React.FC<PaginationProps> = ({ current = 1, pageSize = 10, total = 0, ...props }: PaginationProps) => {
  return (
    <Pagination current={current} pageSize={pageSize} total={total} {...props} />
  );
};

export default PaginationComponent;
