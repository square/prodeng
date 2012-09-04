require 'elvis'

describe 'Elvis' do
  it 'works' do
    became_ruid = nil
    became_euid = nil
    Elvis.run_as("evan") {
      became_ruid = Process.uid
      became_euid = Process.euid
    }
    became_ruid.should == 1326
    became_euid.should == 1326
  end
end

