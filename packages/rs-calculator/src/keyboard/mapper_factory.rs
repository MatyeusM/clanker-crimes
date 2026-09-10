//! Mapper factory. Constructs key mapper strategies. The pipeline has an
//! entry point, and this factory constructs the thing that calls the entry
//! point. Architecture is just delegation all the way down.
use super::mapper_trait::CompositeKeyMapper;

/// Factory for key mapper strategies.
pub struct MapperFactory;

impl MapperFactory {
  /// Creates the composite key mapper (the one true mapper).
  pub fn create_composite() -> CompositeKeyMapper {
    CompositeKeyMapper
  }
}
