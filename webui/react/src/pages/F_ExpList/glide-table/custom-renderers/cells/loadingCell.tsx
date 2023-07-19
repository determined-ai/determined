import { CustomCell, CustomRenderer, GridCellKind } from '@hpe.com/glide-data-grid';

import { rgba2str, rgbaFromGradient, str2rgba } from 'utils/color';
import { Theme } from 'utils/themes';

import { roundedRect } from '../utils';

interface LoadingCellProps {
  readonly appTheme: Theme;
  readonly kind: 'loading-cell';
}

export type LoadingCell = CustomCell<LoadingCellProps>;

const PADDING = 10;
const MAX_STEPS = 10;

const renderer: CustomRenderer<LoadingCell> = {
  draw: (args, cell) => {
    const { ctx, rect, requestAnimationFrame } = args;
    const { appTheme } = cell.data;

    ctx.save();

    const x = rect.x + PADDING;
    const y = rect.y + PADDING;
    const w = rect.width - 2 * PADDING;
    const h = rect.height - 2 * PADDING;

    /**
     * Chart to help figure out all the necessary interval steps to calculate final gradient
     * colors and color stops.
     *
     * percent  inc   factor  color1-stop   color0                color1                color2
     * -----------------------------------------------------------------------------------------------------
     * 0.0      0.0   0       0.0           g(rgba0, rgba1, 0.0)  g(rgba0, rgba1, 0.0)  g(rgba0, rgba1, 1.0)
     * 0.1      0.2   0       0.2           g(rgba0, rgba1, 0.2)  g(rgba0, rgba1, 0.0)  g(rgba0, rgba1, 0.8)
     * 0.25     0.5   0       0.5           g(rgba0, rgba1, 0.5)  g(rgba0, rgba1, 0.0)  g(rgba0, rgba1, 0.5)
     * 0.4      0.8   0       0.8           g(rgba0, rgba1, 0.8)  g(rgba0, rgba1, 0.0)  g(rgba0, rgba1, 0.2)
     * 0.5      0.0   1       0.0           g(rgba0, rgba1, 1.0)  g(rgba0, rgba1, 1.0)  g(rgba0, rgba1, 0.0)
     * 0.6      0.2   1       0.2           g(rgba0, rgba1, 0.8)  g(rgba0, rgba1, 1.0)  g(rgba0, rgba1, 0.2)
     * 0.75     0.5   1       0.5           g(rgba0, rgba1, 0.5)  g(rgba0, rgba1, 1.0)  g(rgba0, rgba1, 0.5)
     * 0.9      0.8   1       0.8           g(rgba0, rgba1, 0.2)  g(rgba0, rgba1, 1.0)  g(rgba0, rgba1, 0.8)
     */

    const step = Math.round(Date.now() / 100) % MAX_STEPS;
    const percent = step / MAX_STEPS;
    const inc = (2 * percent) % 1;
    const factor = percent >= 0.5 ? 1.0 : 0.0;
    const rgba0 = str2rgba(appTheme.ixBorder);
    const rgba1 = str2rgba(appTheme.ixBorderStrong);
    const color0 = rgba2str(rgbaFromGradient(rgba0, rgba1, Math.abs(factor - inc)));
    const color1 = rgba2str(rgbaFromGradient(rgba0, rgba1, factor));
    const color2 = rgba2str(rgbaFromGradient(rgba0, rgba1, 1 - Math.abs(factor - inc)));

    const gradient = ctx.createLinearGradient(x, y, x + w, y);
    gradient.addColorStop(0.0, color0);
    gradient.addColorStop((2 * percent) % 1, color1);
    gradient.addColorStop(1.0, color2);

    ctx.beginPath();
    roundedRect(ctx, x, y, w, h, 2);
    ctx.fillStyle = gradient;
    ctx.fill();

    ctx.restore();

    requestAnimationFrame();
  },
  isMatch: (cell: CustomCell): cell is LoadingCell => {
    return 'kind' in cell.data && cell.data.kind === 'loading-cell';
  },
  kind: GridCellKind.Custom,
  needsHover: true,
  needsHoverPosition: true,
  onPaste: (_v, d) => d,
  provideEditor: () => undefined,
};

export default renderer;
