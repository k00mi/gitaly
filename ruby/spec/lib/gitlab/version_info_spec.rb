require 'spec_helper'

describe Gitlab::VersionInfo do
  let(:unknown) { described_class.new }
  let(:v0_0_1) { described_class.new(0, 0, 1) }
  let(:v0_1_0) { described_class.new(0, 1, 0) }
  let(:v1_0_0) { described_class.new(1, 0, 0) }
  let(:v1_0_1) { described_class.new(1, 0, 1) }
  let(:v1_1_0) { described_class.new(1, 1, 0) }
  let(:v2_0_0) { described_class.new(2, 0, 0) }

  context '>' do
    it { expect(v2_0_0).to be > v1_1_0 }
    it { expect(v1_1_0).to be > v1_0_1 }
    it { expect(v1_0_1).to be > v1_0_0 }
    it { expect(v1_0_0).to be > v0_1_0 }
    it { expect(v0_1_0).to be > v0_0_1 }
  end

  context '>=' do
    it { expect(v2_0_0).to be >= described_class.new(2, 0, 0) }
    it { expect(v2_0_0).to be >= v1_1_0 }
  end

  context '<' do
    it { expect(v0_0_1).to be < v0_1_0 }
    it { expect(v0_1_0).to be < v1_0_0 }
    it { expect(v1_0_0).to be < v1_0_1 }
    it { expect(v1_0_1).to be < v1_1_0 }
    it { expect(v1_1_0).to be < v2_0_0 }
  end

  context '<=' do
    it { expect(v0_0_1).to be <= described_class.new(0, 0, 1) }
    it { expect(v0_0_1).to be <= v0_1_0 }
  end

  context '==' do
    it { expect(v0_0_1).to eq(described_class.new(0, 0, 1)) }
    it { expect(v0_1_0).to eq(described_class.new(0, 1, 0)) }
    it { expect(v1_0_0).to eq(described_class.new(1, 0, 0)) }
  end

  context '!=' do
    it { expect(v0_0_1).not_to eq(v0_1_0) }
  end

  context 'unknown' do
    it { expect(unknown).not_to be v0_0_1 }
    it { expect(unknown).not_to be described_class.new }
    it { expect { unknown > v0_0_1 }.to raise_error(ArgumentError) }
    it { expect { unknown < v0_0_1 }.to raise_error(ArgumentError) }
  end

  context 'parse' do
    it { expect(described_class.parse("1.0.0")).to eq(v1_0_0) }
    it { expect(described_class.parse("1.0.0.1")).to eq(v1_0_0) }
    it { expect(described_class.parse("git 1.0.0b1")).to eq(v1_0_0) }
    it { expect(described_class.parse("git 1.0b1")).not_to be_valid }
  end

  context 'to_s' do
    it { expect(v1_0_0.to_s).to eq("1.0.0") }
    it { expect(unknown.to_s).to eq("Unknown") }
  end
end
