class Range
  # Provide a fake client for use in testing. Ideally a set of tests should be
  # run against both this and the real client to ensure they are sync.
  FakeClient = Struct.new(:responses) do
    def expand(query)
      responses.fetch(query)
    end
  end
end
