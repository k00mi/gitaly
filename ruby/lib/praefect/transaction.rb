module Praefect
  class Transaction
    PRAEFECT_SERVER_METADATA_KEY = "gitaly-praefect-server".freeze
    PRAEFECT_SERVER_PAYLOAD_KEY = "praefect".freeze
    TRANSACTION_METADATA_KEY = "gitaly-reference-transaction".freeze
    TRANSACTION_PAYLOAD_KEY = "transaction".freeze

    MissingPraefectMetadataError = Class.new(StandardError)

    def self.from_metadata(metadata)
      transaction_metadata = metadata[TRANSACTION_METADATA_KEY]
      return new(nil, nil) unless transaction_metadata

      praefect_metadata = metadata[PRAEFECT_SERVER_METADATA_KEY]
      raise MissingPraefectMetadataError, "missing praefect server metadata" unless praefect_metadata

      praefect = JSON.parse(Base64.decode64(praefect_metadata))
      transaction = JSON.parse(Base64.decode64(transaction_metadata))

      new(praefect, transaction)
    end

    def initialize(server, transaction)
      @server = server
      @transaction = transaction
    end

    def payload
      {
        TRANSACTION_PAYLOAD_KEY => @transaction,
        PRAEFECT_SERVER_PAYLOAD_KEY => @server
      }.reject { |_, v| v.nil? }
    end
  end
end
