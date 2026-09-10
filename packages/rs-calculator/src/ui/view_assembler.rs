//! View assembler. Snaps the display plus all seven row factories into the
//! main column. The grand finale of the UI pipeline. Seven imports, one
//! column macro, zero regrets.
use crate::messages::message::Message;
use crate::state::calculator::Calculator;
use crate::ui::column_assembler::{column_alignment, column_padding, column_spacing};
use crate::ui::rows::row_a_factory::build_row_a;
use crate::ui::rows::row_b_factory::build_row_b;
use crate::ui::rows::row_c_factory::build_row_c;
use crate::ui::rows::row_d_factory::build_row_d;
use crate::ui::rows::row_e_factory::build_row_e;
use crate::ui::rows::row_f_factory::build_row_f;
use crate::ui::rows::row_g_factory::build_row_g;
use iced::{Element, widget::column};

/// Builds the full calculator view.
pub fn view(app: &Calculator) -> Element<'_, Message> {
  let display = crate::display::display_factory::create_display(app);
  let row_a = build_row_a(app);
  let row_b = build_row_b(app);
  let row_c = build_row_c(app);
  let row_d = build_row_d(app);
  let row_e = build_row_e(app);
  let row_f = build_row_f(app);
  let row_g = build_row_g(app);

  column![display, row_a, row_b, row_c, row_d, row_e, row_f, row_g,]
    .spacing(column_spacing())
    .padding(column_padding())
    .align_x(column_alignment())
    .into()
}
