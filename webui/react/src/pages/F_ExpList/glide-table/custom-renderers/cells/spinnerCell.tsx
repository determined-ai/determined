import { CustomCell, CustomRenderer, GridCellKind } from '@glideapps/glide-data-grid';

interface SpinnerCellProps {
  readonly kind: 'spinner-cell';
}

export type SpinnerCell = CustomCell<SpinnerCellProps>;

const renderer: CustomRenderer<SpinnerCell> = {
  draw: (args) => {
    const { ctx, rect, requestAnimationFrame } = args;

    const progress = (window.performance.now() % 1000) / 1000;

    const x = rect.x + rect.width / 2;
    const y = rect.y + rect.height / 2;
    ctx.arc(
      x,
      y,
      Math.min(12, rect.height / 6),
      Math.PI * 2 * progress,
      Math.PI * 2 * progress + Math.PI * 1.5,
    );

    ctx.strokeStyle = getComputedStyle(ctx.canvas).getPropertyValue('--theme-brand');
    ctx.lineWidth = 1.5;
    ctx.stroke();

    ctx.lineWidth = 1;

    requestAnimationFrame();

    return true;
  },
  isMatch: (cell: CustomCell): cell is SpinnerCell =>
    (cell.data as SpinnerCellProps).kind === 'spinner-cell',
  kind: GridCellKind.Custom,
  provideEditor: () => undefined,
};

export default renderer;
